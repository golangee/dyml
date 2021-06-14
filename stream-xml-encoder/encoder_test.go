package streamxmlencoder

import (
	"testing"
)

func TestEncoder(t *testing.T) {
	var encoder XMLEncoder
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{
			name: "encoder_test.go",
			text: "#hello@id{world}",
			want: `<root>
			<!-- saying
				 hello world
			
			-->
			<hello>world</hello>
			</root>`,
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output string
			encoder = NewEncoderFromNameAndString(test.name, test.text)
			output, err := encoder.EncodeToXML()

			if test.wantErr {
				if err == nil {
					t.Errorf("expected Error")
				}
			} else {
				if err != nil {
					t.Error(err)
				} else {
					if output != test.want {
						t.Errorf("Test %s failed: %s does not equal %s", test.name, output, test.want)
					}
				}
			}

		})
	}
}
