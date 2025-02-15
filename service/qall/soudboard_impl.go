package qall

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/go-audio/wav"
	"github.com/gofrs/uuid"
	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
)

type soundboardManager struct {
	repo repository.SoundboardRepository
	fs   storage.FileStorage
	l    *zap.Logger
}

func NewSoundboardManager(repo repository.SoundboardRepository, fs storage.FileStorage, logger *zap.Logger) (Soundboard, error) {
	return &soundboardManager{
		repo: repo,
		fs:   fs,
		l:    logger.Named("soundboard_manager"),
	}, nil
}

func (m *soundboardManager) SaveSoundboardItem(soundID uuid.UUID, soundName string, contentType string, fileType model.FileType, src io.Reader, stampID *uuid.UUID, creatorID uuid.UUID) error {
	// ファイルを全読み込み
	b, err := io.ReadAll(src)
	if err != nil {
		m.l.Error("failed to read whole src stream", zap.Error(err))
		return err
	}

	// ファイルの秒数をチェック
	if err := checkAudioDuration(b, contentType, 30); err != nil {
		m.l.Error("failed to check audio duration", zap.Error(err))
		return err
	}

	// ファイルを保存
	err = m.fs.SaveByKey(src, soundID.String(), "soundboardItem", contentType, fileType)
	if err != nil {
		m.l.Error("failed to save soundboard item", zap.Error(err))
		return err
	}

	return m.repo.CreateSoundBoardItem(soundID, soundName, stampID, creatorID)
}

func (m *soundboardManager) GetURL(soundID uuid.UUID) (string, error) {
	return m.fs.GenerateAccessURL(soundID.String(), model.FileTypeSoundboardItem)
}

func (m *soundboardManager) DeleteSoundboardItem(soundID uuid.UUID) error {
	err := m.fs.DeleteByKey(soundID.String(), model.FileTypeSoundboardItem)
	if err != nil {
		m.l.Error("failed to delete soundboard item", zap.Error(err))
		return err
	}

	return m.repo.DeleteSoundboardItem(soundID)
}

// checkAudioDuration は拡張子(ext)に基づいて対応ライブラリを使い、秒数をチェックする
// mp3 / wav / ogg に対応し、それ以外は "we only support mp3, wav, ogg" エラー
func checkAudioDuration(fileBytes []byte, contentType string, maxSeconds float64) error {
	switch contentType {
	case "audio/mpeg", "audio/mp3":
		dur, err := getMp3Duration(fileBytes)
		if err != nil {
			return fmt.Errorf("mp3 decode error: %w", err)
		}
		if dur > maxSeconds {
			return fmt.Errorf("audio is too long (%.1f sec). Must be <= %.0f", dur, maxSeconds)
		}
		return nil

	case "audio/wav", "audio/x-wav":
		dur, err := getWavDuration(fileBytes)
		if err != nil {
			return fmt.Errorf("wav decode error: %w", err)
		}
		if dur > maxSeconds {
			return fmt.Errorf("audio is too long (%.1f sec). Must be <= %.0f", dur, maxSeconds)
		}
		return nil

	case "audio/ogg":
		dur, err := getOggDuration(fileBytes)
		if err != nil {
			return fmt.Errorf("ogg decode error: %w", err)
		}
		if dur > maxSeconds {
			return fmt.Errorf("audio is too long (%.1f sec). Must be <= %.0f", dur, maxSeconds)
		}
		return nil

	default:
		return errors.New("we only support .mp3, .wav, .ogg")
	}
}

// getMp3Duration returns duration in seconds for MP3
func getMp3Duration(data []byte) (float64, error) {
	r := bytes.NewReader(data)
	decoder, err := mp3.NewDecoder(r)
	if err != nil {
		return 0, err
	}
	// decoder.Length() はサンプル数
	// decoder.SampleRate() はサンプリングレート(例: 44100)
	sampleRate := float64(decoder.SampleRate())
	totalSamples := float64(decoder.Length())
	if sampleRate <= 0 {
		return 0, errors.New("invalid mp3 sample rate")
	}
	seconds := totalSamples / sampleRate
	return seconds, nil
}

// getWavDuration returns duration in seconds for WAV
func getWavDuration(data []byte) (float64, error) {
	r := bytes.NewReader(data)
	wavDecoder := wav.NewDecoder(r)
	buf, err := wavDecoder.FullPCMBuffer()
	if err != nil {
		return 0, err
	}
	if buf == nil || buf.Format == nil {
		return 0, errors.New("invalid wav format or buffer")
	}
	sampleRate := float64(buf.Format.SampleRate)
	sampleCount := float64(len(buf.Data)) // PCMBufferのサンプル数
	if sampleRate <= 0 {
		return 0, errors.New("invalid wav sample rate")
	}
	seconds := sampleCount / sampleRate
	return seconds, nil
}

// getOggDuration returns duration in seconds for OGG(Vorbis)
func getOggDuration(data []byte) (float64, error) {
	r := bytes.NewReader(data)
	stream, err := oggvorbis.NewReader(r)
	if err != nil {
		return 0, err
	}
	sampleRate := float64(stream.SampleRate())
	sampleCount := float64(stream.Length()) // Total samples
	if sampleRate <= 0 {
		return 0, errors.New("invalid ogg sample rate")
	}
	seconds := sampleCount / sampleRate
	return seconds, nil
}
