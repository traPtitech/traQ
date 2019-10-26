package extension

import (
	"github.com/labstack/echo/v4"
)

var defaultBinder echo.DefaultBinder

// Binder echo.Binderのカスタム
type Binder struct{}

// Bind encoding/jsonをjsoniter.ConfigCompatibleWithStandardLibraryに置換
func (b *Binder) Bind(i interface{}, c echo.Context) error {
	return defaultBinder.Bind(i, c)
	/* TODO 戻す
	req := c.Request()
	if req.ContentLength == 0 || !strings.HasPrefix(req.Header.Get(echo.HeaderContentType), echo.MIMEApplicationJSON) {
		return defaultBinder.Bind(i, c)
	}

	if err := jsoniter.ConfigFastest.NewDecoder(req.Body).Decode(i); err != nil {
		if ute, ok := err.(*json.UnmarshalTypeError); ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
		} else if se, ok := err.(*json.SyntaxError); ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	return nil
	*/
}
