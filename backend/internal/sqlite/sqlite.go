package sqlite

/*
#cgo linux LDFLAGS: -lsqlite3
#cgo freebsd CFLAGS: -I/usr/local/include
#cgo freebsd LDFLAGS: -L/usr/local/lib -lsqlite3
#include <stdlib.h>
#include <sqlite3.h>
static int samba_bind_text(sqlite3_stmt* stmt, int idx, const char* value, int n) { return sqlite3_bind_text(stmt, idx, value, n, SQLITE_TRANSIENT); }
static int samba_bind_blob(sqlite3_stmt* stmt, int idx, const void* value, int n) { return sqlite3_bind_blob(stmt, idx, value, n, SQLITE_TRANSIENT); }
*/
import "C"

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type DB struct {
	mu sync.Mutex
	db *C.sqlite3
}

type Result struct{ RowsAffected int64 }
type Rows struct {
	db     *DB
	stmt   *C.sqlite3_stmt
	closed bool
	err    error
}
type Row struct {
	values []any
	err    error
}
type NullString struct {
	String string
	Valid  bool
}

var ErrNoRows = errors.New("no rows")

func Open(path string) (*DB, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var db *C.sqlite3
	flags := C.int(C.SQLITE_OPEN_READWRITE | C.SQLITE_OPEN_CREATE | C.SQLITE_OPEN_FULLMUTEX)
	if rc := C.sqlite3_open_v2(cpath, &db, flags, nil); rc != C.SQLITE_OK {
		msg := "sqlite open failed"
		if db != nil {
			msg = C.GoString(C.sqlite3_errmsg(db))
			C.sqlite3_close(db)
		}
		return nil, errors.New(msg)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return nil
	}
	if rc := C.sqlite3_close(d.db); rc != C.SQLITE_OK {
		return fmt.Errorf("sqlite close: %d", rc)
	}
	d.db = nil
	return nil
}
func (d *DB) Ping() error { _, err := d.Exec("SELECT 1"); return err }
func (d *DB) ExecScript(script string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	cs := C.CString(script)
	defer C.free(unsafe.Pointer(cs))
	var errMsg *C.char
	if rc := C.sqlite3_exec(d.db, cs, nil, nil, &errMsg); rc != C.SQLITE_OK {
		msg := "sqlite error"
		if errMsg != nil {
			msg = C.GoString(errMsg)
			C.sqlite3_free(unsafe.Pointer(errMsg))
		}
		return errors.New(msg)
	}
	return nil
}
func (d *DB) Exec(query string, args ...any) (Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	stmt, err := prepare(d.db, query)
	if err != nil {
		return Result{}, err
	}
	defer C.sqlite3_finalize(stmt)
	if err = bindAll(stmt, args); err != nil {
		return Result{}, err
	}
	for {
		rc := C.sqlite3_step(stmt)
		switch rc {
		case C.SQLITE_DONE:
			return Result{RowsAffected: int64(C.sqlite3_changes(d.db))}, nil
		case C.SQLITE_ROW:
			continue
		default:
			return Result{}, fmt.Errorf("sqlite step: %s", C.GoString(C.sqlite3_errmsg(d.db)))
		}
	}
}
func (d *DB) Query(query string, args ...any) (*Rows, error) {
	d.mu.Lock()
	stmt, err := prepare(d.db, query)
	if err != nil {
		d.mu.Unlock()
		return nil, err
	}
	if err = bindAll(stmt, args); err != nil {
		C.sqlite3_finalize(stmt)
		d.mu.Unlock()
		return nil, err
	}
	return &Rows{db: d, stmt: stmt}, nil
}
func (d *DB) QueryRow(query string, args ...any) *Row {
	rows, err := d.Query(query, args...)
	if err != nil {
		return &Row{err: err}
	}
	defer rows.Close()
	if !rows.Next() {
		if rows.Err() != nil {
			return &Row{err: rows.Err()}
		}
		return &Row{err: ErrNoRows}
	}
	return &Row{values: rows.current()}
}
func (r *Rows) Next() bool {
	if r.closed || r.err != nil {
		return false
	}
	rc := C.sqlite3_step(r.stmt)
	if rc == C.SQLITE_ROW {
		return true
	}
	if rc != C.SQLITE_DONE {
		r.err = fmt.Errorf("sqlite step: %s", C.GoString(C.sqlite3_errmsg(r.db.db)))
	}
	return false
}
func (r *Rows) Err() error             { return r.err }
func (r *Rows) Scan(dest ...any) error { return assign(r.current(), dest) }
func (r *Rows) current() []any {
	n := int(C.sqlite3_column_count(r.stmt))
	v := make([]any, n)
	for i := 0; i < n; i++ {
		switch C.sqlite3_column_type(r.stmt, C.int(i)) {
		case C.SQLITE_INTEGER:
			v[i] = int64(C.sqlite3_column_int64(r.stmt, C.int(i)))
		case C.SQLITE_FLOAT:
			v[i] = float64(C.sqlite3_column_double(r.stmt, C.int(i)))
		case C.SQLITE_TEXT:
			p := C.sqlite3_column_text(r.stmt, C.int(i))
			v[i] = C.GoString((*C.char)(unsafe.Pointer(p)))
		case C.SQLITE_BLOB:
			p := C.sqlite3_column_blob(r.stmt, C.int(i))
			n := C.sqlite3_column_bytes(r.stmt, C.int(i))
			v[i] = C.GoBytes(p, n)
		default:
			v[i] = nil
		}
	}
	return v
}
func (r *Rows) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	C.sqlite3_finalize(r.stmt)
	r.db.mu.Unlock()
	return nil
}
func (r *Row) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return assign(r.values, dest)
}

func prepare(db *C.sqlite3, query string) (*C.sqlite3_stmt, error) {
	cq := C.CString(query)
	defer C.free(unsafe.Pointer(cq))
	var stmt *C.sqlite3_stmt
	if rc := C.sqlite3_prepare_v2(db, cq, -1, &stmt, nil); rc != C.SQLITE_OK {
		return nil, errors.New(C.GoString(C.sqlite3_errmsg(db)))
	}
	return stmt, nil
}
func bindAll(stmt *C.sqlite3_stmt, args []any) error {
	for i, arg := range args {
		idx := C.int(i + 1)
		var rc C.int
		switch v := arg.(type) {
		case nil:
			rc = C.sqlite3_bind_null(stmt, idx)
		case string:
			cs := C.CString(v)
			rc = C.samba_bind_text(stmt, idx, cs, C.int(len(v)))
			C.free(unsafe.Pointer(cs))
		case []byte:
			if len(v) == 0 {
				rc = C.samba_bind_blob(stmt, idx, nil, 0)
			} else {
				rc = C.samba_bind_blob(stmt, idx, unsafe.Pointer(&v[0]), C.int(len(v)))
			}
		case int:
			rc = C.sqlite3_bind_int64(stmt, idx, C.sqlite3_int64(v))
		case int64:
			rc = C.sqlite3_bind_int64(stmt, idx, C.sqlite3_int64(v))
		case bool:
			if v {
				rc = C.sqlite3_bind_int(stmt, idx, 1)
			} else {
				rc = C.sqlite3_bind_int(stmt, idx, 0)
			}
		case float64:
			rc = C.sqlite3_bind_double(stmt, idx, C.double(v))
		case time.Time:
			text := v.UTC().Format(time.RFC3339Nano)
			cs := C.CString(text)
			rc = C.samba_bind_text(stmt, idx, cs, C.int(len(text)))
			C.free(unsafe.Pointer(cs))
		default:
			return fmt.Errorf("unsupported sqlite bind type %T", arg)
		}
		if rc != C.SQLITE_OK {
			return fmt.Errorf("sqlite bind failed: %d", rc)
		}
	}
	return nil
}
func assign(values []any, dest []any) error {
	if len(values) != len(dest) {
		return fmt.Errorf("scan count mismatch: %d != %d", len(values), len(dest))
	}
	for i := range dest {
		if err := set(dest[i], values[i]); err != nil {
			return fmt.Errorf("column %d: %w", i, err)
		}
	}
	return nil
}
func set(dst, src any) error {
	switch d := dst.(type) {
	case *string:
		if src == nil {
			*d = ""
		} else {
			*d = fmt.Sprint(src)
		}
		return nil
	case *int:
		if src == nil {
			*d = 0
			return nil
		}
		switch v := src.(type) {
		case int64:
			*d = int(v)
		case float64:
			*d = int(v)
		case string:
			n, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			*d = n
		}
		return nil
	case *int64:
		if src == nil {
			*d = 0
			return nil
		}
		if v, ok := src.(int64); ok {
			*d = v
			return nil
		}
	case *float64:
		if src == nil {
			*d = 0
			return nil
		}
		if v, ok := src.(float64); ok {
			*d = v
			return nil
		}
	case *NullString:
		if src == nil {
			d.String = ""
			d.Valid = false
		} else {
			d.String = fmt.Sprint(src)
			d.Valid = true
		}
		return nil
	case *[]byte:
		if v, ok := src.([]byte); ok {
			*d = v
			return nil
		}
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("destination must be non-nil pointer")
	}
	if src == nil {
		rv.Elem().Set(reflect.Zero(rv.Elem().Type()))
		return nil
	}
	sv := reflect.ValueOf(src)
	if sv.Type().ConvertibleTo(rv.Elem().Type()) {
		rv.Elem().Set(sv.Convert(rv.Elem().Type()))
		return nil
	}
	return fmt.Errorf("cannot assign %T to %T", src, dst)
}
