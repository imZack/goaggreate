package jq

// #cgo LDFLAGS: -l jq
// #include "jq.h"
// #include <stdlib.h>
/*
static void on_jq_err(void *cx, jv err) {
	if (*(const char**)cx != NULL) {
		return; // only capture the first error.
	}
	*(const char**)cx = jv_string_value(err);
}

static void set_err_cb(jq_state *jq, const char **msg) {
	if (msg == NULL) {
		jq_set_error_cb(jq, NULL, NULL);
		return;
	}
	jq_set_error_cb(jq, on_jq_err, msg);
}

static const char* get_jv_error(jv value) {
	if (jv_is_valid(value)) return NULL;

	value = jv_invalid_get_msg(jv_copy(value));
	if (jv_get_kind(value) != JV_KIND_STRING) return NULL;

	return jv_string_value(value);
}
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"
	"unsafe"
)

// Apply compiles a JQ filter, and applies it to one or more inputs.
func Apply(proc string, input ...interface{}) ([][]byte, error) {
	vm, err := Compile(proc)
	defer vm.Close()
	if err != nil {
		return nil, err
	}
	seq := make([][]byte, 0, len(input)*4)
	for _, inp := range input {
		js, err := json.Marshal(inp)
		if err != nil {
			return seq, err
		}
		for ret := range vm.Run(js, &err) {
			seq = append(seq, ret)
		}
		if err != nil {
			return seq, err
		}
	}
	return seq, nil
}

// Compile compiles a JQ filter into a new JQ virtual machine.
func Compile(proc string) (*Vm, error) {
	s := new(Vm)
	s.jq = C.jq_init()

	err := compileJq(s.jq, proc)
	if err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// A Vm encloses the internal state of a compiled JQ filter machine.  Vm's
// can be reused, but cannot be used concurrently.
type Vm struct {
	lock sync.Mutex
	jq   *C.jq_state
}

// Apply2 ...
func (s *Vm) Apply2(input string) (string, error) {
	var result string

	if s.jq == nil {
		return result, errors.New("null jq instance")
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	cinput := C.CString(input)
	defer C.free(unsafe.Pointer(cinput))
	inputData := C.jv_parse(cinput)
	if !isValid(inputData) {
		return result, jvError(inputData)
	}

	C.jq_start(s.jq, inputData, 0)

	var v C.jv
	for {
		v = C.jq_next(s.jq)
		if C.jv_is_valid(v) == 0 {
			break
		}
		//defer C.jv_free(v)

		str := C.jv_dump_string(v, 0)
		defer C.jv_free(str)
		result = C.GoString(C.jv_string_value(str))
	}
	if err := jvError(v); err != nil {
		return result, err
	}
	return result, nil
}

// Apply runs the JQ filter on each input after using encoding/json to convert to JSON.
// The results of each run are combined into an array of JSON raw messages.
// Apply stops on the first error, which could be during Compile or a Run.
func (s *Vm) Apply(input interface{}) ([][]byte, error) {
	if s.jq == nil {
		return [][]byte{}, errors.New("null jq instance")
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	seq := make([][]byte, 0, 4)

	js, err := json.Marshal(input)
	if err != nil {
		return seq, err
	}
	for ret := range s.Run(js, &err) {
		seq = append(seq, ret)
	}
	if err != nil {
		return seq, err
	}

	return seq, nil
}

// Run starts the filter with the supplied input, and uses a channel to gather results.
// When the channel is closed, e will contain the final error, if any.
// Run may be used consecutively for additional inputs, but not in parallel.
func (s *Vm) Run(input []byte, e *error) chan []byte {
	out := make(chan []byte)
	if len(input) < 1 {
		close(out)
		return out
	}

	jv := jvParse(input)
	if !isValid(jv) {
		provideError(e, jvError(jv))
		close(out)
		return out
	}

	go func() {
		defer close(out)
		//TODO: if invalid in, report an error
		err := processJq(s.jq, jv, func(val C.jv) {
			defer freeJv(val)
			next := dumpJv(val)
			if len(next) > 0 {
				out <- next
			}
		})
		provideError(e, err)
	}()
	return out
}

func provideError(e *error, err error) {
	switch {
	case e == nil:
		return
	case err == nil:
		return
	}
	*e = err
}

// Close closes a JQ state, releasing resources.
func (s *Vm) Close() error {
	if s == nil {
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	C.jq_teardown(&s.jq)
	s.jq = nil
	return nil
}

// starts JQ with a value, and visits each result; the value must be freeJv'd by the visitor.
func processJq(jq *C.jq_state, input C.jv, visit func(val C.jv)) error {
	var jv C.jv
	C.jq_start(jq, input, 0)
	for {
		jv = C.jq_next(jq)
		if !isValid(jv) {
			break
		}
		visit(jv)
	}
	defer freeJv(jv)
	return jvError(jv)
}

// identifies valid JQ values; JQ produces invalids for errors
func isValid(jv C.jv) bool {
	return C.jv_is_valid(jv) != 0
}

// compileJq compiles a program into the jq interpreter
func compileJq(jq *C.jq_state, src string) error {
	var msg *C.char
	C.set_err_cb(jq, &msg)
	defer C.set_err_cb(jq, nil)
	csrc := C.CString(src)
	defer C.free(unsafe.Pointer(csrc))
	// TODO: use a SyntaxError type to split up Error from Expr
	if C.jq_compile(jq, csrc) == 0 {
		return errors.New(C.GoString(msg))
	}
	return nil
}

// freeJv releases a jq value; JQ does not release outputs or inputs; you must do this on its behalf.
func freeJv(jv C.jv) {
	C.jv_free(jv)
}

// jvParse parses a new jq value from a string; you must defer a freeJv to release the result
func jvParse(p []byte) C.jv {
	if len(p) < 1 {
		panic("zero length or nil input")
	}
	return C.jv_parse_sized(
		(*C.char)(unsafe.Pointer(&p[0])),
		C.int(len(p)))
}

// if jv is invalid, and references an error message, return a proper Go error
func jvError(jv C.jv) error {
	ptr := C.get_jv_error(jv)
	if ptr == nil {
		return nil
	}
	return errors.New(C.GoString(ptr))
}

// the builtin jv_dump_term (and jv_dump) is extremely inefficient, repeatedly using strlen / strcat to
// build a string
func dumpJv(jv C.jv) []byte {
	var buf bytes.Buffer
	dumpValue(&buf, jv)
	return buf.Bytes()
}

func dumpValue(buf *bytes.Buffer, jv C.jv) {
	switch C.jv_get_kind(jv) {
	case C.JV_KIND_NULL:
		buf.WriteString("null")
	case C.JV_KIND_TRUE:
		buf.WriteString("true")
	case C.JV_KIND_FALSE:
		buf.WriteString("false")
	case C.JV_KIND_NUMBER:
		d := float64(C.jv_number_value(jv))
		p, err := json.Marshal(d)
		if err != nil {
			panic(err)
		}
		buf.Write(p)
	case C.JV_KIND_ARRAY:
		dumpArray(buf, jv)
	case C.JV_KIND_OBJECT:
		dumpObject(buf, jv)
	case C.JV_KIND_STRING:
		dumpString(buf, jv)
	default:
		panic(int(C.jv_get_kind(jv)))
	}
}

func dumpObject(buf *bytes.Buffer, x C.jv) {
	keys := C.jv_keys(C.jv_copy(x))
	defer C.jv_free(keys)
	ct := C.jv_array_length(C.jv_copy(keys))

	buf.WriteRune('{')
	defer buf.WriteRune('}')
	for i := C.int(0); i < ct; i++ {
		key := C.jv_array_get(C.jv_copy(keys), i)
		if i > 0 {
			buf.WriteRune(',')
		}
		dumpString(buf, key)
		buf.WriteRune(':')
		val := C.jv_object_get(C.jv_copy(x), key)
		dumpValue(buf, val)
		C.jv_free(val)
	}
}

func dumpArray(buf *bytes.Buffer, x C.jv) {
	ct := C.jv_array_length(C.jv_copy(x))

	buf.WriteRune('[')
	defer buf.WriteRune(']')
	for i := C.int(0); i < ct; i++ {
		val := C.jv_array_get(C.jv_copy(x), i)
		if i > 0 {
			buf.WriteRune(',')
		}
		dumpValue(buf, val)
		C.jv_free(val)
	}
}

func dumpString(buf *bytes.Buffer, x C.jv) {
	ptr := C.jv_string_value(x)
	ct := C.jv_string_length_bytes(C.jv_copy(x))
	p, err := json.Marshal(C.GoStringN(ptr, ct))
	if err != nil {
		panic(err)
	}
	buf.Write(p)
}
