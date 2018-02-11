package model

import (
	"github.com/satori/go.uuid"
	"testing"
)

func TestDevice_TableName(t *testing.T) {
	dev := &Device{}
	if "devices" != dev.TableName() {
		t.Fatalf("tablename is wrong:want devices, actual %s", dev.TableName())
	}
}

func TestDevice_Register(t *testing.T) {
	beforeTest(t)
	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"

	dev := &Device{UserId: id1, Token: token1}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id2, Token: token2}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id1, Token: token2}
	if err := dev.Register(); err == nil {
		t.Fatalf("device register doesn't fail")
	}

	if l, _ := db.Count(&Device{}); l != 2 {
		t.Fatalf("registered device count is wrong: want 2, actual %v", l)
	}

}

func TestDevice_Unregister(t *testing.T) {
	beforeTest(t)
	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	dev := &Device{UserId: id1, Token: token1}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id2, Token: token2}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id1, Token: token3}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}

	target := &Device{Token: token2}
	if err := target.Unregister(); err != nil {
		t.Fatalf("device unregister failed: %v", err)
	}
	if l, _ := db.Count(&Device{}); l != 2 {
		t.Fatalf("registered device count is wrong: want 2, actual %v", l)
	}

	target = &Device{UserId: id1}
	if err := target.Unregister(); err != nil {
		t.Fatalf("device unregister failed: %v", err)
	}
	if l, _ := db.Count(&Device{}); l != 0 {
		t.Fatalf("registered device count is wrong: want 0, actual %v", l)
	}

}

func TestGetAllDevices(t *testing.T) {
	beforeTest(t)
	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	dev := &Device{UserId: id1, Token: token1}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id2, Token: token2}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id1, Token: token3}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}

	devs, err := GetAllDevices()
	if err != nil {
		t.Fatalf("failed to get all devices: %v", err)
	}
	if len(devs) != 3 {
		t.Fatalf("the number of devices is wrong: want 3, actual %v", len(devs))
	}
}

func TestGetDevices(t *testing.T) {
	beforeTest(t)
	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	dev := &Device{UserId: id1, Token: token1}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id2, Token: token2}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}
	dev = &Device{UserId: id1, Token: token3}
	if err := dev.Register(); err != nil {
		t.Fatalf("device register failed: %v", err)
	}

	devs, err := GetDevices(uuid.FromStringOrNil(id1))
	if err != nil {
		t.Fatalf("failed to get devices: %v", err)
	}
	if len(devs) != 2 {
		t.Fatalf("the number of devices is wrong: want 2, actual %v", len(devs))
	}

	devs, err = GetDevices(uuid.FromStringOrNil(id2))
	if err != nil {
		t.Fatalf("failed to get devices: %v", err)
	}
	if len(devs) != 1 {
		t.Fatalf("the number of devices is wrong: want 1, actual %v", len(devs))
	}

}
