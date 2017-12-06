package notification

import (
	"github.com/sideshow/apns2"
	"os"
)

var (
	APN_CERTIFICATE_FILE = os.Getenv("APN_CERTIFICATE_FILE")
	APNClient            apns2.Client
)

func init() {
	//TODO Initialize APN Client
}
