package consulparser

import (
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestNewParser(t *testing.T) {
	type args struct {
		client func() *api.Client
	}
	generalClient, _ := api.NewClient(api.DefaultConfig())
	tests := []struct {
		name       string
		args       args
		wantParser func() ParserIface
		wantErr    bool
	}{
		{
			name: "Initialize Parser",
			args: args{
				client: func() *api.Client {
					return generalClient
				},
			},
			wantParser: func() ParserIface {
				parser := &Parser{
					consulKV: generalClient.KV(),
				}
				return parser
			},
			wantErr: false,
		},
		{
			name: "Nil Client",
			args: args{
				client: func() *api.Client {
					return nil
				},
			},
			wantParser: func() ParserIface {
				return nil
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotParser, err := NewParser(tt.args.client())
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValues(t, tt.wantParser(), gotParser)
		})
	}
}

func TestParser_Parse(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	const (
		responseJSON = `[
				{
					"LockIndex": 0,
					"Key": "%s",
					"Flags": 0,
					"Value": "%s",
					"CreateIndex": 0,
					"ModifyIndex": 0
				}
			]
		`
	)
	// Init response mock for consul client
	stringResp := fmt.Sprintf(responseJSON, "string", base64.StdEncoding.EncodeToString([]byte("hello")))
	intResp := fmt.Sprintf(responseJSON, "integer", base64.StdEncoding.EncodeToString([]byte("-10")))
	floatResp := fmt.Sprintf(responseJSON, "float", base64.StdEncoding.EncodeToString([]byte("10.0")))
	uintResp := fmt.Sprintf(responseJSON, "unsignedinteger", base64.StdEncoding.EncodeToString([]byte("1000")))
	boolResp := fmt.Sprintf(responseJSON, "boolean", base64.StdEncoding.EncodeToString([]byte("true")))
	timeResp := fmt.Sprintf(responseJSON, "time", base64.StdEncoding.EncodeToString([]byte("2019-02-01T00:00:00Z")))
	overflowIntResp := fmt.Sprintf(responseJSON, "overflowint", base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(math.MaxInt8+1, 10))))
	overflowUintResp := fmt.Sprintf(responseJSON, "overflowuint", base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(math.MaxUint8+1, 10))))
	overflowFloatResp := fmt.Sprintf(responseJSON, "overflowfloat", base64.StdEncoding.EncodeToString([]byte(strconv.FormatFloat(math.MaxFloat64-100, 'E', 0, 64))))
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/string",
		httpmock.NewStringResponder(http.StatusOK, stringResp),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/integer",
		httpmock.NewStringResponder(http.StatusOK, intResp),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/float",
		httpmock.NewStringResponder(http.StatusOK, floatResp),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/unsignedinteger",
		httpmock.NewStringResponder(http.StatusOK, uintResp),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/boolean",
		httpmock.NewStringResponder(http.StatusOK, boolResp),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/time",
		httpmock.NewStringResponder(http.StatusOK, timeResp),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/overflowint",
		httpmock.NewStringResponder(http.StatusOK, overflowIntResp),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/overflowuint",
		httpmock.NewStringResponder(http.StatusOK, overflowUintResp),
	)

	httpmock.RegisterResponder(
		http.MethodGet,
		"http://127.0.0.1:8500/v1/kv/overflowfloat",
		httpmock.NewStringResponder(http.StatusOK, overflowFloatResp),
	)
	type fields struct {
		consulKV func() *api.KV
	}
	type args struct {
		target interface{}
	}
	type MLPtrPartStruct struct {
		Float           float64     `consulkv:"float"`
		UnsignedInteger uint64      `consulkv:"unsignedinteger"`
		Boolean         bool        `consulkv:"boolean"`
		Interface       interface{} `consulkv:"string"`
	}
	tests := []struct {
		name         string
		fields       fields
		args         func() args
		wantErr      bool
		expectResult func() interface{}
	}{
		{
			name: "Normal Struct Case",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						String          string      `consulkv:"string"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				return &struct {
					String          string      `consulkv:"string"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{
					String:          "hello",
					Integer:         -10,
					Float:           10.0,
					UnsignedInteger: 1000,
					Boolean:         true,
					Interface:       "hello",
				}
			},
		},
		{
			name: "Recursive Struct in Field of Struct Case",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						String     string `consulkv:"string"`
						Integer    int64  `consulkv:"integer"`
						PartStruct struct {
							Float           float64     `consulkv:"float"`
							UnsignedInteger uint64      `consulkv:"unsignedinteger"`
							Boolean         bool        `consulkv:"boolean"`
							Interface       interface{} `consulkv:"string"`
						}
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				return &struct {
					String     string `consulkv:"string"`
					Integer    int64  `consulkv:"integer"`
					PartStruct struct {
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}
				}{
					String:  "hello",
					Integer: -10,
					PartStruct: struct {
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{
						Float:           10.0,
						UnsignedInteger: 1000,
						Boolean:         true,
						Interface:       "hello",
					},
				}
			},
		},
		{
			name: "Non Pointer Type",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: struct {
						String          string      `consulkv:"string"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return struct {
					String          string      `consulkv:"string"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{}
			},
		},
		{
			name: "Unsupported Type",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						String          map[string]string `consulkv:"string"`
						Integer         int64             `consulkv:"integer"`
						Float           float64           `consulkv:"float"`
						UnsignedInteger uint64            `consulkv:"unsignedinteger"`
						Boolean         bool              `consulkv:"boolean"`
						Interface       interface{}       `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					String          map[string]string `consulkv:"string"`
					Integer         int64             `consulkv:"integer"`
					Float           float64           `consulkv:"float"`
					UnsignedInteger uint64            `consulkv:"unsignedinteger"`
					Boolean         bool              `consulkv:"boolean"`
					Interface       interface{}       `consulkv:"string"`
				}{}
			},
		},
		{
			name: "Key not found in first field",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						String          string      `consulkv:"string/hello"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					String          string      `consulkv:"string/hello"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{}
			},
		},
		{
			name: "Key not found in nth field",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						String          string      `consulkv:"hello"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"hello/float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					String          string      `consulkv:"hello"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"hello/float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{}
			},
		},
		{
			name: "Key is not valid",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						String          string      `consulkv:"hello"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"/unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					String          string      `consulkv:"hello"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"/unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{}
			},
		},
		{
			name: "Private field",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						text            string      `consulkv:"string"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				return &struct {
					text            string      `consulkv:"string"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{
					text:            "",
					Integer:         -10,
					Float:           10.0,
					UnsignedInteger: 1000,
					Boolean:         true,
					Interface:       "hello",
				}
			},
		},
		{
			name: "Bad Assignment for field type integer",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Text            string      `consulkv:"string"`
						Integer         int64       `consulkv:"string"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					Text            string      `consulkv:"string"`
					Integer         int64       `consulkv:"string"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{
					Text: "hello",
				}
			},
		},
		{
			name: "Bad Assignment for field type float",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Text            string      `consulkv:"string"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"string"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					Text            string      `consulkv:"string"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"string"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{
					Text:    "hello",
					Integer: -10,
				}
			},
		},
		{
			name: "Bad Assignment for field type unsigned integer",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Text            string      `consulkv:"string"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"string"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					Text            string      `consulkv:"string"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"string"`
					Boolean         bool        `consulkv:"boolean"`
					Interface       interface{} `consulkv:"string"`
				}{
					Text:    "hello",
					Integer: -10,
					Float:   10.0,
				}
			},
		},
		{
			name: "Bad Assignment for field type boolean",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Text            string      `consulkv:"string"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"string"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					Text            string      `consulkv:"string"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"string"`
					Interface       interface{} `consulkv:"string"`
				}{
					Text:            "hello",
					Integer:         -10,
					Float:           10.0,
					UnsignedInteger: 1000,
				}
			},
		},
		{
			name: "Case for String Pointer in Struct Field",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Text            *string     `consulkv:"string"`
						Integer         int64       `consulkv:"integer"`
						Float           float64     `consulkv:"float"`
						UnsignedInteger uint64      `consulkv:"unsignedinteger"`
						Boolean         bool        `consulkv:"boolean"`
						Interface       interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				stringPointer := "hello"
				return &struct {
					Text            *string     `consulkv:"string"`
					Integer         int64       `consulkv:"integer"`
					Float           float64     `consulkv:"float"`
					UnsignedInteger uint64      `consulkv:"unsignedinteger"`
					Boolean         bool        `consulkv:"string"`
					Interface       interface{} `consulkv:"string"`
				}{
					Text:            &stringPointer,
					Integer:         -10,
					Float:           10.0,
					UnsignedInteger: 1000,
					Boolean:         true,
					Interface:       "hello",
				}
			},
		},
		{
			name: "Case for All Type Pointer in Struct Field",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Text            *string      `consulkv:"string"`
						Integer         *int64       `consulkv:"integer"`
						Float           *float64     `consulkv:"float"`
						UnsignedInteger *uint64      `consulkv:"unsignedinteger"`
						Boolean         *bool        `consulkv:"boolean"`
						Interface       *interface{} `consulkv:"string"`
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				stringPointer := "hello"
				intPointer := int64(-10)
				floatPointer := float64(10.0)
				boolPointer := true
				uintPointer := uint64(1000)
				interfacePointer := (interface{})("hello")
				return &struct {
					Text            *string      `consulkv:"string"`
					Integer         *int64       `consulkv:"integer"`
					Float           *float64     `consulkv:"float"`
					UnsignedInteger *uint64      `consulkv:"unsignedinteger"`
					Boolean         *bool        `consulkv:"string"`
					Interface       *interface{} `consulkv:"string"`
				}{
					Text:            &stringPointer,
					Integer:         &intPointer,
					Float:           &floatPointer,
					UnsignedInteger: &uintPointer,
					Boolean:         &boolPointer,
					Interface:       &interfacePointer,
				}
			},
		},
		{
			name: "Case for Recursive Pointer Struct",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Text      *string `consulkv:"string"`
						Integer   *int64  `consulkv:"integer"`
						Recursive *struct {
							Float           *float64     `consulkv:"float"`
							UnsignedInteger *uint64      `consulkv:"unsignedinteger"`
							Boolean         *bool        `consulkv:"boolean"`
							Interface       *interface{} `consulkv:"string"`
						}
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				stringPointer := "hello"
				intPointer := int64(-10)
				floatPointer := float64(10.0)
				boolPointer := true
				uintPointer := uint64(1000)
				interfacePointer := (interface{})("hello")
				return &struct {
					Text      *string `consulkv:"string"`
					Integer   *int64  `consulkv:"integer"`
					Recursive *struct {
						Float           *float64     `consulkv:"float"`
						UnsignedInteger *uint64      `consulkv:"unsignedinteger"`
						Boolean         *bool        `consulkv:"boolean"`
						Interface       *interface{} `consulkv:"string"`
					}
				}{
					Text:    &stringPointer,
					Integer: &intPointer,
					Recursive: &struct {
						Float           *float64     `consulkv:"float"`
						UnsignedInteger *uint64      `consulkv:"unsignedinteger"`
						Boolean         *bool        `consulkv:"boolean"`
						Interface       *interface{} `consulkv:"string"`
					}{
						Float:           &floatPointer,
						UnsignedInteger: &uintPointer,
						Boolean:         &boolPointer,
						Interface:       &interfacePointer,
					},
				}
			},
		},
		{
			name: "Case for time.Time and *time.Time in Struct Field",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						TimeVal     time.Time  `consulkv:"time"`
						TimePointer *time.Time `consulkv:"time"`
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				timeParsed, err := time.Parse(time.RFC3339, "2019-02-01T00:00:00Z")
				if err != nil {
					t.Errorf("Error in parsing the time value: %s", err)
				}
				return &struct {
					TimeVal     time.Time  `consulkv:"time"`
					TimePointer *time.Time `consulkv:"time"`
				}{
					TimeVal:     timeParsed,
					TimePointer: &timeParsed,
				}
			},
		},
		{
			name: "Case Overflow Integer",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						OverflowInt int8 `consulkv:"overflowint"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					OverflowInt int8 `consulkv:"overflowint"`
				}{}
			},
		},
		{
			name: "Case Overflow Unsigned Integer",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						OverflowUint uint8 `consulkv:"overflowuint"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					OverflowUint uint8 `consulkv:"overflowuint"`
				}{}
			},
		},
		{
			name: "Case Overflow Float",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						OverflowFloat float32 `consulkv:"overflowfloat"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					OverflowFloat float32 `consulkv:"overflowfloat"`
				}{}
			},
		},
		{
			name: "Case Overflow Integer Pointer",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						OverflowInt *int8 `consulkv:"overflowint"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					OverflowInt *int8 `consulkv:"overflowint"`
				}{}
			},
		},
		{
			name: "Case Overflow Unsigned Integer Pointer",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						OverflowUint *uint8 `consulkv:"overflowuint"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					OverflowUint *uint8 `consulkv:"overflowuint"`
				}{}
			},
		},
		{
			name: "Case Overflow Float Pointer",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						OverflowFloat *float32 `consulkv:"overflowfloat"`
					}{},
				}
			},
			wantErr: true,
			expectResult: func() interface{} {
				return &struct {
					OverflowFloat *float32 `consulkv:"overflowfloat"`
				}{}
			},
		},
		{
			name: "1-Level Pointer in field struct",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Integer **int64 `consulkv:"integer"`
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				integerNum := int64(-10)
				integerNumPtr := &integerNum
				return &struct {
					Integer **int64 `consulkv:"integer"`
				}{
					Integer: &integerNumPtr,
				}
			},
		},
		{
			name: "2 or More Level Pointer in field struct",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				return args{
					target: &struct {
						Integer ***int64 `consulkv:"integer"`
					}{},
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				integerNum := int64(-10)
				integerNumPtr := &integerNum
				integerNumPtr1 := &integerNumPtr
				return &struct {
					Integer ***int64 `consulkv:"integer"`
				}{
					Integer: &integerNumPtr1,
				}
			},
		},
		{
			name: "Multi Level Pointer Target Case",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				level1 := &struct {
					String     string `consulkv:"string"`
					Integer    int64  `consulkv:"integer"`
					PartStruct ***MLPtrPartStruct
				}{}
				level2 := &level1
				level3 := &level2
				return args{
					target: level3,
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				level1 := &MLPtrPartStruct{
					Float:           10.0,
					UnsignedInteger: 1000,
					Boolean:         true,
					Interface:       "hello",
				}
				level2 := &level1
				level1Target := &struct {
					String     string `consulkv:"string"`
					Integer    int64  `consulkv:"integer"`
					PartStruct ***MLPtrPartStruct
				}{
					String:     "hello",
					Integer:    -10,
					PartStruct: &level2,
				}
				level2Target := &level1Target
				level3Target := &level2Target
				return level3Target
			},
		},
		{
			name: "Multi Level Nil Pointer Target Case",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(&api.Config{
						HttpClient: &http.Client{},
					})
					if err != nil {
						t.Error("Failed to start the client!")
					}
					return client.KV()
				},
			},
			args: func() args {
				level1 := (interface{})(nil)
				level2 := &level1
				level3 := &level2
				return args{
					target: &level3,
				}
			},
			wantErr: false,
			expectResult: func() interface{} {
				level1 := (interface{})(nil)
				level2 := &level1
				level3 := &level2
				return &level3
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &Parser{
				consulKV: tt.fields.consulKV(),
			}
			target := tt.args().target
			if err := parser.Parse(target); (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.EqualValues(t, tt.expectResult(), target)
		})
	}
}

func TestParser_SetTimeLayout(t *testing.T) {
	type fields struct {
		consulKV func() *api.KV
	}
	type args struct {
		layout func() string
	}
	type expects struct {
		layout func() string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expects expects
		wantErr bool
	}{
		{
			name: "time.RFC3339Nano Layout",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(api.DefaultConfig())
					if err != nil {
						t.Errorf("Error starting the api client: %s", err)
					}
					return client.KV()
				},
			},
			args: args{
				layout: func() string {
					return time.RFC3339Nano
				},
			},
			expects: expects{
				layout: func() string {
					return time.RFC3339Nano
				},
			},
			wantErr: false,
		},
		{
			name: "Empty Layout",
			fields: fields{
				consulKV: func() *api.KV {
					client, err := api.NewClient(api.DefaultConfig())
					if err != nil {
						t.Errorf("Error starting the api client: %s", err)
					}
					return client.KV()
				},
			},
			args: args{
				layout: func() string {
					return ""
				},
			},
			expects: expects{
				layout: func() string {
					return time.RFC3339
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				timeLayout = time.RFC3339
			}()
			parser := &Parser{
				consulKV: tt.fields.consulKV(),
			}
			if err := parser.SetTimeLayout(tt.args.layout()); (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.expects.layout(), tt.args.layout())
			}
			assert.Equal(t, tt.expects.layout(), timeLayout)
		})
	}
}
