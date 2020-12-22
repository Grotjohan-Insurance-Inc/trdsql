package trdsql

// Convert JSON to a table.
// Supports the following JSON container types.
// * Array ([{c1: 1}, {c1: 2}, {c1: 3}])
// * Multiple JSON ({c1: 1}\n {c1: 2}\n {c1: 3}\n)

import (
	"encoding/json"
	"errors"
	"io"
	"log"
)

// JSONReader provides methods of the Reader interface.
type JSONReader struct {
	reader  *json.Decoder
	preRead []map[string]string
	names   []string
	types   []string
	inArray []interface{}
	count   int
}

// NewJSONReader returns JSONReader and error.
func NewJSONReader(reader io.Reader, opts *ReadOpts) (*JSONReader, error) {
	r := &JSONReader{}
	r.reader = json.NewDecoder(reader)
	var top interface{}
	names := map[string]bool{}
	for i := 0; i < opts.InPreRead; i++ {
		row, keys, err := r.readAhead(top, i)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return r, err
			}
			debug.Printf(err.Error())
			return r, nil
		}
		r.preRead = append(r.preRead, row)
		for k := 0; k < len(keys); k++ {
			if !names[keys[k]] {
				names[keys[k]] = true
				r.names = append(r.names, keys[k])
			}
		}
	}

	return r, nil
}

// Names returns column names.
func (r *JSONReader) Names() ([]string, error) {
	return r.names, nil
}

// Types returns column types.
// All JSON types return the DefaultDBType.
func (r *JSONReader) Types() ([]string, error) {
	r.types = make([]string, len(r.names))
	for i := 0; i < len(r.names); i++ {
		r.types[i] = DefaultDBType
	}
	return r.types, nil
}

func (r *JSONReader) readAhead(top interface{}, count int) (map[string]string, []string, error) {
	if r.inArray != nil {
		if len(r.inArray) > count {
			r.count++
			return r.secondLevel(top, r.inArray[count])
		}
		return nil, nil, io.EOF
	}
	err := r.reader.Decode(&top)
	if err != nil {
		return nil, nil, err
	}
	return r.topLevel(top)
}

func (r *JSONReader) topLevel(top interface{}) (map[string]string, []string, error) {
	switch obj := top.(type) {
	case []interface{}:
		// [{} or [] or etc...]
		r.inArray = obj
		return r.secondLevel(top, r.inArray[0])
	case map[string]interface{}:
		// {"a":"b"} object
		r.inArray = nil
		return r.objectFirstRow(obj)
	}
	return nil, nil, ErrUnableConvert
}

// Analyze second when top is array.
func (r *JSONReader) secondLevel(top interface{}, second interface{}) (map[string]string, []string, error) {
	switch obj := second.(type) {
	case map[string]interface{}:
		// [{}]
		return r.objectFirstRow(obj)
	case []interface{}:
		// [[]]
		return r.etcFirstRow(second)
	default:
		// ["a","b"]
		r.inArray = nil
		return r.etcFirstRow(top)
	}
}

func (r *JSONReader) objectFirstRow(obj map[string]interface{}) (map[string]string, []string, error) {
	// {"a":"b"} object
	name := make([]string, 0, len(obj))
	row := make(map[string]string)
	for k, v := range obj {
		name = append(name, k)
		row[k] = jsonString(v)
	}
	return row, name, nil
}

func (r *JSONReader) etcFirstRow(val interface{}) (map[string]string, []string, error) {
	// ex. array array
	// [["a"],
	//  ["b"]]
	var name []string
	k := "c1"
	name = append(name, k)
	row := make(map[string]string)
	row[k] = jsonString(val)
	return row, name, nil
}

func jsonString(val interface{}) string {
	switch val.(type) {
	case map[string]interface{}, []interface{}:
		str, err := json.Marshal(val)
		if err != nil {
			log.Printf("ERROR: jsonString:%s", err)
		}
		return ValString(str)
	default:
		return ValString(val)
	}
}

// PreReadRow is returns only columns that store preread rows.
func (r *JSONReader) PreReadRow() [][]interface{} {
	rowNum := len(r.preRead)
	rows := make([][]interface{}, rowNum)
	for n := 0; n < rowNum; n++ {
		rows[n] = make([]interface{}, len(r.names))
		for i := range r.names {
			rows[n][i] = r.preRead[n][r.names[i]]
		}
	}
	return rows
}

// ReadRow is read the rest of the row.
func (r *JSONReader) ReadRow(row []interface{}) ([]interface{}, error) {
	if r.inArray != nil {
		// [] array
		r.count++
		if r.count >= len(r.inArray) {
			var top interface{}
			err := r.reader.Decode(&top)
			if err != nil {
				return nil, err
			}
			r.count = 0
		}
		if len(r.inArray) > 0 {
			row = r.rowParse(row, r.inArray[r.count])
		}
		return row, nil
	}

	// {} object
	var data interface{}
	err := r.reader.Decode(&data)
	if err != nil {
		return nil, err
	}
	row = r.rowParse(row, data)
	return row, nil
}

func (r *JSONReader) rowParse(row []interface{}, jsonRow interface{}) []interface{} {
	switch m := jsonRow.(type) {
	case map[string]interface{}:
		for i := range r.names {
			row[i] = jsonString(m[r.names[i]])
		}
	default:
		for i := range r.names {
			row[i] = nil
		}
		row[0] = jsonString(jsonRow)
	}
	return row
}
