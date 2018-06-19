package router

import "testing"

func TestLoadWebhookTemplate(t *testing.T) {
	LoadWebhookTemplate("../static/webhook/*.tmpl")
}
