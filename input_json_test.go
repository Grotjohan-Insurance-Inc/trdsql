package trdsql

import (
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func arraySortEqual(t *testing.T, a []string, b []string) bool {
	t.Helper()
	if len(a) != len(b) {
		return false
	}

	copyA := make([]string, len(a))
	copyB := make([]string, len(b))

	copy(copyA, a)
	copy(copyB, b)

	sort.Strings(copyA)
	sort.Strings(copyB)

	return reflect.DeepEqual(copyA, copyB)
}

func TestNewJSONReader(t *testing.T) {
	type args struct {
		reader io.Reader
		opts   *ReadOpts
	}
	tests := []struct {
		name    string
		args    args
		want    *JSONReader
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				reader: strings.NewReader(""),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   nil,
				preRead: nil,
			},
			wantErr: false,
		},
		{
			name: "invalidJSON",
			args: args{
				reader: strings.NewReader("t"),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   nil,
				preRead: nil,
			},
			wantErr: true,
		},
		{
			name: "emptyJSON",
			args: args{
				reader: strings.NewReader("{}"),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   nil,
				preRead: []map[string]string{{}},
			},
			wantErr: false,
		},
		{
			name: "test1",
			args: args{
				reader: strings.NewReader(`[{"c1":"1","c2":"Orange"},{"c1":"2","c2":"Melon"},{"c1":"3","c2":"Apple"}]`),
				opts:   NewReadOpts(InPreRead(3)),
			},
			want: &JSONReader{
				names:   []string{"c1", "c2"},
				preRead: []map[string]string{{"c1": "1", "c2": "Orange"}, {"c1": "2", "c2": "Melon"}, {"c1": "3", "c2": "Apple"}},
			},
			wantErr: false,
		},
		{
			name: "test2",
			args: args{
				reader: strings.NewReader(`
{"c1":"1","c2":"Orange"}
{"c1":"2","c2":"Melon"}
{"c1":"3","c2":"Apple"}`),
				opts: NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1", "c2"},
				preRead: []map[string]string{{"c1": "1", "c2": "Orange"}},
			},
			wantErr: false,
		},
		{
			name: "testArray",
			args: args{
				reader: strings.NewReader(`[["a"],["b"]]`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1"},
				preRead: []map[string]string{{"c1": "[\"a\"]"}, {"c1": "[\"b\"]"}},
			},
			wantErr: false,
		},
		{
			name: "testArray2",
			args: args{
				reader: strings.NewReader(`[["a","b"],["c","d"]]`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1"},
				preRead: []map[string]string{{"c1": "[\"a\",\"b\"]"}, {"c1": "[\"c\",\"d\"]"}},
			},
			wantErr: false,
		},
		{
			name: "testArray3",
			args: args{
				reader: strings.NewReader(`["a","b"]`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"c1"},
				preRead: []map[string]string{{"c1": "a"}, {"c1": "b"}},
			},
			wantErr: false,
		},
		{
			name: "testObject",
			args: args{
				reader: strings.NewReader(`{"a":"b"}`),
				opts:   NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"a"},
				preRead: []map[string]string{{"a": "b"}},
			},
			wantErr: false,
		},
		{
			name: "diffColumn",
			args: args{
				reader: strings.NewReader(`
{"id":"1","name":"Orange"}
{"id":"2","name":"Melon"}
{"id":"3","name":"Apple"}
{"id":"4","name":"Banana","color":"Yellow"}`),
				opts: NewReadOpts(),
			},
			want: &JSONReader{
				names:   []string{"id", "name"},
				preRead: []map[string]string{{"id": "1", "name": "Orange"}},
			},
			wantErr: false,
		},
		{
			name: "diffColumn2",
			args: args{
				reader: strings.NewReader(`
{"id":"1","name":"Orange"}
{"id":"2","name":"Melon"}
{"id":"3","name":"Apple"}
{"id":"4","name":"Banana","color":"Yellow"}`),
				opts: NewReadOpts(InPreRead(5)),
			},
			want: &JSONReader{
				names: []string{"id", "name", "color"},
				preRead: []map[string]string{
					{"id": "1", "name": "Orange"},
					{"id": "2", "name": "Melon"},
					{"id": "3", "name": "Apple"},
					{"id": "4", "name": "Banana", "color": "Yellow"},
				},
			},
			wantErr: false,
		},
		{
			name: "testPath",
			args: args{
				reader: strings.NewReader(`[{"c1":"1","c2":"Orange"},{"c1":"2","c2":"Melon"},{"c1":"3","c2":"Apple"}]`),
				opts:   NewReadOpts(InPath("0")),
			},
			want: &JSONReader{
				names:   []string{"c1", "c2"},
				preRead: []map[string]string{{"c1": "1", "c2": "Orange"}},
			},
			wantErr: false,
		},
		{
			name: "testPath2",
			args: args{
				reader: strings.NewReader(`{"employees":[
					{"name":"Shyam", "email":"shyamjaiswal@gmail.com"},
					{"name":"Bob", "email":"bob32@gmail.com"},
					{"name":"Jai", "email":"jai87@gmail.com"}
				]}`),
				opts: NewReadOpts(InPath("employees")),
			},
			want: &JSONReader{
				names: []string{"name", "email"},
				preRead: []map[string]string{
					{"name": "Shyam", "email": "shyamjaiswal@gmail.com"},
					{"name": "Bob", "email": "bob32@gmail.com"},
					{"name": "Jai", "email": "jai87@gmail.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "testPath3",
			args: args{
				reader: strings.NewReader(`{"menu": {
					"id": "file",
					"value": "File",
					"popup": {
					  "menuitem": [
						{"value": "New", "onclick": "CreateDoc()"},
						{"value": "Open", "onclick": "OpenDoc()"},
						{"value": "Save", "onclick": "SaveDoc()"}
					  ]
					}
				  }}`),
				opts: NewReadOpts(InPath("menu.popup.menuitem")),
			},
			want: &JSONReader{
				names: []string{"value", "onclick"},
				preRead: []map[string]string{
					{"value": "New", "onclick": "CreateDoc()"},
					{"value": "Open", "onclick": "OpenDoc()"},
					{"value": "Save", "onclick": "SaveDoc()"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewJSONReader(tt.args.reader, tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewJSONReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !arraySortEqual(t, got.names, tt.want.names) {
				t.Errorf("NewJSONReader() = %v, want %v", got.names, tt.want.names)
			}
			if !reflect.DeepEqual(got.preRead, tt.want.preRead) {
				t.Errorf("NewJSONReader() = %v, want %v", got.preRead, tt.want.preRead)
			}
		})
	}
}
