package session

import (
	"encoding/binary"
	"errors"
	"gogame/base/logger"
	"io"
)

const (
	kSizeMax = 1 << 20 //  1MB
)

var (
	errCodecReadyBodyTooLarge = errors.New("session: Codec ready body too large")
	errCodecWriteBodyTooLarge = errors.New("session: Codec write body too large")
)

type SessionCodec struct {
}

var DefaultSessionCodec = &SessionCodec{}

func (this *SessionCodec) Read(r io.Reader) ([]byte, error) {
	h := make([]byte, 4)
	if _, err := io.ReadFull(r, h); err != nil {
		logger.Info("session: Codec read header error %v", err)
		return nil, err
	}
	n := binary.BigEndian.Uint32(h)
	if n > kSizeMax {
		logger.Error(errCodecReadyBodyTooLarge.Error())
		return nil, errCodecReadyBodyTooLarge
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		logger.Info("session: Codec read body error %v", err)
		return nil, err
	}
	return b, nil
}

func (this *SessionCodec) Write(w io.Writer, b []byte) error {
	n := uint32(len(b))
	if n > kSizeMax {
		logger.Error(errCodecWriteBodyTooLarge.Error())
		return errCodecWriteBodyTooLarge
	}
	h := make([]byte, 4)
	binary.BigEndian.PutUint32(h, n)
	if _, err := w.Write(h); err != nil {
		logger.Info("session: Codec write header error %v", err)
		return err
	}
	if _, err := w.Write(b); err != nil {
		logger.Info("session: Codec write body error %v", err)
		return err
	}
	return nil
}
