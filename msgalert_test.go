// Copyright (c) 2013 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcwire_test

import (
	"bytes"
	"github.com/conformal/btcwire"
	"github.com/davecgh/go-spew/spew"
	"io"
	"reflect"
	"testing"
)

// TestAlert tests the MsgAlert API.
func TestAlert(t *testing.T) {
	pver := btcwire.ProtocolVersion
	payloadblob := "some message"
	signature := "some sig"

	// Ensure we get the same payload and signature back out.
	msg := btcwire.NewMsgAlert(payloadblob, signature)
	if msg.PayloadBlob != payloadblob {
		t.Errorf("NewMsgAlert: wrong payloadblob - got %v, want %v",
			msg.PayloadBlob, payloadblob)
	}
	if msg.Signature != signature {
		t.Errorf("NewMsgAlert: wrong signature - got %v, want %v",
			msg.Signature, signature)
	}

	// Ensure the command is expected value.
	wantCmd := "alert"
	if cmd := msg.Command(); cmd != wantCmd {
		t.Errorf("NewMsgAlert: wrong command - got %v want %v",
			cmd, wantCmd)
	}

	// Ensure max payload is expected value.
	wantPayload := uint32(1024 * 1024 * 32)
	maxPayload := msg.MaxPayloadLength(pver)
	if maxPayload != wantPayload {
		t.Errorf("MaxPayloadLength: wrong max payload length for "+
			"protocol version %d - got %v, want %v", pver,
			maxPayload, wantPayload)
	}

	return
}

// TestAlertWire tests the MsgAlert wire encode and decode for various protocol
// versions.
func TestAlertWire(t *testing.T) {
	baseAlert := btcwire.NewMsgAlert("some payload", "somesig")
	baseAlertEncoded := []byte{
		0x0c, // Varint for payload length
		0x73, 0x6f, 0x6d, 0x65, 0x20, 0x70, 0x61, 0x79,
		0x6c, 0x6f, 0x61, 0x64, // "some payload"
		0x07,                                     // Varint for signature length
		0x73, 0x6f, 0x6d, 0x65, 0x73, 0x69, 0x67, // "somesig"
	}

	tests := []struct {
		in   *btcwire.MsgAlert // Message to encode
		out  *btcwire.MsgAlert // Expected decoded message
		buf  []byte            // Wire encoding
		pver uint32            // Protocol version for wire encoding
	}{
		// Latest protocol version.
		{
			baseAlert,
			baseAlert,
			baseAlertEncoded,
			btcwire.ProtocolVersion,
		},

		// Protocol version BIP0035Version.
		{
			baseAlert,
			baseAlert,
			baseAlertEncoded,
			btcwire.BIP0035Version,
		},

		// Protocol version BIP0031Version.
		{
			baseAlert,
			baseAlert,
			baseAlertEncoded,
			btcwire.BIP0031Version,
		},

		// Protocol version NetAddressTimeVersion.
		{
			baseAlert,
			baseAlert,
			baseAlertEncoded,
			btcwire.NetAddressTimeVersion,
		},

		// Protocol version MultipleAddressVersion.
		{
			baseAlert,
			baseAlert,
			baseAlertEncoded,
			btcwire.MultipleAddressVersion,
		},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode the message to wire format.
		var buf bytes.Buffer
		err := test.in.BtcEncode(&buf, test.pver)
		if err != nil {
			t.Errorf("BtcEncode #%d error %v", i, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), test.buf) {
			t.Errorf("BtcEncode #%d\n got: %s want: %s", i,
				spew.Sdump(buf.Bytes()), spew.Sdump(test.buf))
			continue
		}

		// Decode the message from wire format.
		var msg btcwire.MsgAlert
		rbuf := bytes.NewBuffer(test.buf)
		err = msg.BtcDecode(rbuf, test.pver)
		if err != nil {
			t.Errorf("BtcDecode #%d error %v", i, err)
			continue
		}
		if !reflect.DeepEqual(&msg, test.out) {
			t.Errorf("BtcDecode #%d\n got: %s want: %s", i,
				spew.Sdump(msg), spew.Sdump(test.out))
			continue
		}
	}
}

// TestAlertWireErrors performs negative tests against wire encode and decode
// of MsgAlert to confirm error paths work correctly.
func TestAlertWireErrors(t *testing.T) {
	pver := btcwire.ProtocolVersion

	baseAlert := btcwire.NewMsgAlert("some payload", "somesig")
	baseAlertEncoded := []byte{
		0x0c, // Varint for payload length
		0x73, 0x6f, 0x6d, 0x65, 0x20, 0x70, 0x61, 0x79,
		0x6c, 0x6f, 0x61, 0x64, // "some payload"
		0x07,                                     // Varint for signature length
		0x73, 0x6f, 0x6d, 0x65, 0x73, 0x69, 0x67, // "somesig"
	}

	tests := []struct {
		in       *btcwire.MsgAlert // Value to encode
		buf      []byte            // Wire encoding
		pver     uint32            // Protocol version for wire encoding
		max      int               // Max size of fixed buffer to induce errors
		writeErr error             // Expected write error
		readErr  error             // Expected read error
	}{
		// Force error in payload length.
		{baseAlert, baseAlertEncoded, pver, 0, io.ErrShortWrite, io.EOF},
		// Force error in payload.
		{baseAlert, baseAlertEncoded, pver, 1, io.ErrShortWrite, io.EOF},
		// Force error in signature length.
		{baseAlert, baseAlertEncoded, pver, 13, io.ErrShortWrite, io.EOF},
		// Force error in signature.
		{baseAlert, baseAlertEncoded, pver, 14, io.ErrShortWrite, io.EOF},
	}

	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		// Encode to wire format.
		w := newFixedWriter(test.max)
		err := test.in.BtcEncode(w, test.pver)
		if reflect.TypeOf(err) != reflect.TypeOf(test.writeErr) {
			t.Errorf("BtcEncode #%d wrong error got: %v, want: %v",
				i, err, test.writeErr)
			continue
		}

		// For errors which are not of type btcwire.MessageError, check
		// them for equality.
		if _, ok := err.(*btcwire.MessageError); !ok {
			if err != test.writeErr {
				t.Errorf("BtcEncode #%d wrong error got: %v, "+
					"want: %v", i, err, test.writeErr)
				continue
			}
		}

		// Decode from wire format.
		var msg btcwire.MsgAlert
		r := newFixedReader(test.max, test.buf)
		err = msg.BtcDecode(r, test.pver)
		if reflect.TypeOf(err) != reflect.TypeOf(test.readErr) {
			t.Errorf("BtcDecode #%d wrong error got: %v, want: %v",
				i, err, test.readErr)
			continue
		}

		// For errors which are not of type btcwire.MessageError, check
		// them for equality.
		if _, ok := err.(*btcwire.MessageError); !ok {
			if err != test.readErr {
				t.Errorf("BtcDecode #%d wrong error got: %v, "+
					"want: %v", i, err, test.readErr)
				continue
			}
		}
	}
}
