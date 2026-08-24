package token

import "testing"

func TestSignVerify(t *testing.T) {
	p := Payload{Kind: "o", TenantID: "t1", Campaign: "c1", Recipient: "r1"}
	tok := Sign("secret", p)
	got, err := Verify("secret", tok)
	if err != nil {
		t.Fatal(err)
	}
	if got.Recipient != "r1" {
		t.Fatal(got)
	}
	if _, err := Verify("other", tok); err == nil {
		t.Fatal("forged")
	}
	if _, err := Verify("secret", "aaa.bbb"); err == nil {
		t.Fatal("garbage")
	}
}

func TestOpenRedirectDeniedByKind(t *testing.T) {
	p := Payload{Kind: "c", TenantID: "t", Campaign: "c", Recipient: "r", URL: "https://good.example"}
	tok := Sign("s", p)
	got, err := Verify("s", tok)
	if err != nil || got.URL != "https://good.example" {
		t.Fatal(got, err)
	}
}
