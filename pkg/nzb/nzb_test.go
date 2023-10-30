package nzb

import (
	"bytes"
	_ "embed"
	"io"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

//go:embed nzbmock.xml
var nzbmock []byte

func TestNzbParser_Parse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		buf io.Reader
	}

	tests := []struct {
		name         string
		args         args
		wantNzb      *Nzb
		wantErr      bool
		wantErrorMsg string
	}{
		{
			name: "successful parse",
			args: args{
				buf: bytes.NewBufferString(`
                    <?xml version="1.0" encoding="UTF-8"?>
                    <!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">
                    <nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">
                        <head>
                            <meta type="title">Test NZB</meta>
                        </head>
                        <file poster="test@example.com" date="1234567890" subject="Test File">
                            <groups>
                                <group>alt.binaries.test</group>
                            </groups>
                            <segments>
                                <segment bytes="100" number="1">abc123</segment>
                            </segments>
                        </file>
                    </nzb>
                `),
			},
			wantNzb: &Nzb{
				Meta: map[string]string{
					"title": "Test NZB",
				},
				Files: []*NzbFile{
					{
						Groups: []string{"alt.binaries.test"},
						Segments: []*NzbSegment{
							{
								Bytes:  100,
								Number: 1,
								Id:     "abc123",
							},
						},
						Poster:  "test@example.com",
						Date:    1234567890,
						Subject: "Test File",
					},
				},
			},
			wantErr:      false,
			wantErrorMsg: "",
		},
		{
			name: "failed parse",
			args: args{
				buf: bytes.NewBufferString(`
                    <?xml version="1.0" encoding="UTF-8"?>
                    <!DOCTYPE nzb PUBLIC "-//newzBin//DTD NZB 1.1//EN" "http://www.newzbin.com/DTD/nzb/nzb-1.1.dtd">
                    <nzb xmlns="http://www.newzbin.com/DTD/2003/nzb">
                        <head>
                            <meta type="title">Test NZB</meta>
                        </head>
                        <file poster="test@example.com" date="1234567890" subject="Test File">
                            <groups>
                                <group>alt.binaries.test</group>
                            </groups>
                            <segments>
                                <segment bytes="100" number="1">abc123</segment>
                            </segments>
                        </file>
                `),
			},
			wantNzb:      nil,
			wantErr:      true,
			wantErrorMsg: "XML syntax error on line 16: unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNzb, err := ParseFromBuffer(tt.args.buf)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErrorMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantNzb, gotNzb)
			}
		})
	}
}

func TestNzb_ToBytes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nzb, err := ParseFromBuffer(bytes.NewBuffer(nzbmock))
	assert.NoError(t, err)

	b, err := nzb.ToBytes()
	assert.NoError(t, err)

	assert.Contains(t, string(b), `<meta type="file_size">1442682314</meta>`)
	assert.Contains(t, string(b), `<segment bytes="792581" number="6">NdTkKlQbLxUfOlDfGmFtBdEd-1695410374703@nyuu</segment>`)
}
