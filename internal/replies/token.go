// Package replies lets a campaign see who replied to it. A tracked
// campaign's Reply-To is swapped for a per-message address carrying a
// signed token (tenant, campaign, subscriber); the inbound mail provider
// (Postmark) posts each reply to cmd/replies.go's webhook, which decodes
// the token, records the reply and forwards it to the campaign's real
// Reply-To. See nerd-zero/listnun-console#159.
package replies

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// macLen is how many bytes of the HMAC-SHA256 the token keeps. 7 bytes
// (56 bits) still makes forging a token for another campaign infeasible
// (each guess is a webhook round trip), and is the most that fits: an
// email local part is capped at 64 characters, Postmark's default inbound
// address already uses 33 of them ("<32-hex-hash>+"), and the largest IDs
// take 12 bytes -- 19 bytes base32-encode to 31 characters. 8 bytes made
// the worst case 65 (TestAddressFitsLocalPartLimit).
const macLen = 7

// minSecretLen is the shortest Codec secret accepted -- a short secret
// would make the truncated MAC above guessable.
const minSecretLen = 16

// encoding is lowercase-safe base32 without padding: mail clients and
// servers may change the case of an address's local part, so Decode
// upper-cases before decoding.
var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// ErrInvalidToken is returned by Decode for a token that's malformed or
// whose signature doesn't match -- a forged or corrupted reply address.
var ErrInvalidToken = errors.New("invalid reply token")

// Ref identifies the message a reply answers. SubscriberID is 0 for a
// test send, which has no subscriber.
type Ref struct {
	TenantID     int
	CampaignID   int
	SubscriberID int
}

// Codec signs and verifies reply tokens with a server secret.
type Codec struct {
	key []byte
}

// NewCodec returns a Codec using secret, which must be at least
// minSecretLen characters.
func NewCodec(secret string) (*Codec, error) {
	if len(secret) < minSecretLen {
		return nil, fmt.Errorf("reply token secret must be at least %d characters", minSecretLen)
	}
	return &Codec{key: []byte(secret)}, nil
}

// Encode returns the token for r: its three IDs as uvarints followed by a
// truncated HMAC over them, base32-encoded in lowercase.
func (c *Codec) Encode(r Ref) string {
	buf := make([]byte, 0, 3*binary.MaxVarintLen64+macLen)
	buf = binary.AppendUvarint(buf, uint64(r.TenantID))
	buf = binary.AppendUvarint(buf, uint64(r.CampaignID))
	buf = binary.AppendUvarint(buf, uint64(r.SubscriberID))
	buf = append(buf, c.mac(buf)...)
	return strings.ToLower(encoding.EncodeToString(buf))
}

// Decode verifies token and returns the Ref it carries.
func (c *Codec) Decode(token string) (Ref, error) {
	raw, err := encoding.DecodeString(strings.ToUpper(strings.TrimSpace(token)))
	if err != nil || len(raw) <= macLen {
		return Ref{}, ErrInvalidToken
	}
	payload, sig := raw[:len(raw)-macLen], raw[len(raw)-macLen:]
	if !hmac.Equal(sig, c.mac(payload)) {
		return Ref{}, ErrInvalidToken
	}

	var ids [3]uint64
	rest := payload
	for i := range ids {
		v, n := binary.Uvarint(rest)
		if n <= 0 {
			return Ref{}, ErrInvalidToken
		}
		ids[i], rest = v, rest[n:]
	}
	if len(rest) != 0 || ids[0] == 0 || ids[1] == 0 {
		return Ref{}, ErrInvalidToken
	}
	return Ref{TenantID: int(ids[0]), CampaignID: int(ids[1]), SubscriberID: int(ids[2])}, nil
}

func (c *Codec) mac(payload []byte) []byte {
	h := hmac.New(sha256.New, c.key)
	h.Write(payload)
	return h.Sum(nil)[:macLen]
}

// Addresser builds per-message reply addresses from the inbound address
// the provider gives us, by plus-addressing the token onto its local part:
// "abc@inbound.postmarkapp.com" becomes "abc+<token>@inbound.postmarkapp.com".
// The provider reports the part after "+" back as the MailboxHash.
type Addresser struct {
	codec  *Codec
	local  string
	domain string
}

// NewAddresser returns an Addresser for inboundAddress ("local@domain").
func NewAddresser(codec *Codec, inboundAddress string) (*Addresser, error) {
	local, domain, ok := strings.Cut(strings.TrimSpace(inboundAddress), "@")
	if !ok || local == "" || domain == "" || strings.Contains(local, "+") {
		return nil, fmt.Errorf("reply inbound address %q must look like local@domain with no + in it", inboundAddress)
	}
	return &Addresser{codec: codec, local: local, domain: domain}, nil
}

// Address returns the reply address for r.
func (a *Addresser) Address(r Ref) string {
	return a.local + "+" + a.codec.Encode(r) + "@" + a.domain
}

// Codec returns the Codec the Addresser signs tokens with, for decoding
// the replies it receives.
func (a *Addresser) Codec() *Codec {
	return a.codec
}
