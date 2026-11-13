package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// New tenants copied tenant 1's media storage with no way to give them
// their own. The settings written must be exactly the upload.* keys the
// media store reads, with defaults filled in and bad input refused.
func TestOperatorMediaReq_ToSettings(t *testing.T) {
	set, msg := operatorMediaReq{Provider: "s3", S3: &operatorS3{
		URL: "https://s3.eu-west-1.amazonaws.com/", AccessKeyID: "AK", SecretAccessKey: "SK",
		Region: "eu-west-1", Bucket: "listnun-media", BucketPath: "tenants/42",
	}}.toSettings()
	if msg != "" {
		t.Fatalf("valid S3 refused: %s", msg)
	}
	b, _ := json.Marshal(set)
	for _, want := range []string{
		`"upload.provider":"s3"`, `"upload.s3.url":"https://s3.eu-west-1.amazonaws.com"`,
		`"upload.s3.bucket":"listnun-media"`, `"upload.s3.bucket_path":"/tenants/42"`,
		`"upload.s3.bucket_type":"public"`, `"upload.s3.expiry":"167h"`, `"upload.s3.aws_secret_access_key":"SK"`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("settings %s lack %s", b, want)
		}
	}

	fs, msg := operatorMediaReq{Provider: "filesystem"}.toSettings()
	if msg != "" || fs.Provider != "filesystem" || fs.S3SecretKey != "" || fs.S3AccessKeyID != "" {
		t.Errorf("filesystem = %+v, %q; want the provider set and no S3 credentials left", fs, msg)
	}

	for name, r := range map[string]operatorMediaReq{
		"unknown provider": {Provider: "gcs"},
		"s3 without s3":    {Provider: "s3"},
		"no bucket":        {Provider: "s3", S3: &operatorS3{URL: "u", AccessKeyID: "a", SecretAccessKey: "s"}},
		"bad bucket type":  {Provider: "s3", S3: &operatorS3{URL: "u", Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s", BucketType: "secret"}},
		"bad expiry":       {Provider: "s3", S3: &operatorS3{URL: "u", Bucket: "b", AccessKeyID: "a", SecretAccessKey: "s", Expiry: "a week"}},
	} {
		if _, msg := r.toSettings(); msg == "" {
			t.Errorf("%s: accepted", name)
		}
	}
}
