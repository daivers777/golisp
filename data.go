// Copyright 2014 SteelSeries ApS.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This package implements a basic LISP interpretor for embedding in a go program for scripting.
// This file implements data elements.

package golisp

import (
	"errors"
	"fmt"
	"gopkg.in/fatih/set.v0"
	"strings"
	"unsafe"
)

const (
	ConsCellType = iota
	AlistType
	AlistCellType
	IntegerType
	FloatType
	BooleanType
	StringType
	SymbolType
	FunctionType
	MacroType
	PrimitiveType
	BoxedObjectType
	FrameType
)

type ConsCell struct {
	Car *Data
	Cdr *Data
}

type BoxedObject struct {
	ObjType string
	Obj     unsafe.Pointer
}

type Data struct {
	Type  uint8
	Value unsafe.Pointer
}

// Boolean constants

type BooleanBox struct {
	B bool
}

var b_true *BooleanBox = &BooleanBox{B: true}
var b_false *BooleanBox = &BooleanBox{B: false}

var LispTrue *Data = &Data{Type: BooleanType, Value: unsafe.Pointer(b_true)}
var LispFalse *Data = &Data{Type: BooleanType, Value: unsafe.Pointer(b_false)}

// Debug support

var EvalDepth int = 0
var DebugSingleStep bool = false
var DebugCurrentFrame *SymbolTableFrame = nil
var DebugEvalInDebugRepl bool = false
var DebugErrorEnv *SymbolTableFrame = nil
var DebugOnError bool = false
var IsInteractive bool = false
var DebugReturnValue *Data = nil
var DebugOnEntry *set.Set = set.New()

func TypeOf(d *Data) uint8 {
	return d.Type
}

func TypeName(t uint8) string {
	switch t {
	case ConsCellType:
		return "List"
	case AlistType:
		return "Association List"
	case AlistCellType:
		return "Association List Cell"
	case IntegerType:
		return "Integer"
	case FloatType:
		return "Float"
	case BooleanType:
		return "Boolean"
	case StringType:
		return "String"
	case SymbolType:
		return "Symbol"
	case FunctionType:
		return "Function"
	case MacroType:
		return "Macro"
	case PrimitiveType:
		return "Primitive"
	case FrameType:
		return "Frame"
	case BoxedObjectType:
		return "Go Object"
	default:
		return "Unknown"
	}
}

func NilP(d *Data) bool {
	if d == nil {
		return true
	}
	if (PairP(d) || AlistP(d) || DottedPairP(d)) && Car(d) == nil && Cdr(d) == nil {
		return true
	}
	return false
}

func NotNilP(d *Data) bool {
	return !NilP(d)
}

func PairP(d *Data) bool {
	return d == nil || TypeOf(d) == ConsCellType
}

func ListP(d *Data) bool {
	return PairP(d) || AlistP(d)
}

func DottedPairP(d *Data) bool {
	return d == nil || TypeOf(d) == AlistCellType
}

func AlistP(d *Data) bool {
	return d == nil || TypeOf(d) == AlistType
}

func BooleanP(d *Data) bool {
	return d != nil && TypeOf(d) == BooleanType
}

func SymbolP(d *Data) bool {
	return d != nil && TypeOf(d) == SymbolType
}

func NakedP(d *Data) bool {
	return d != nil && TypeOf(d) == SymbolType && strings.HasSuffix(StringValue(d), ":")
}

func StringP(d *Data) bool {
	return d != nil && TypeOf(d) == StringType
}

func IntegerP(d *Data) bool {
	return d != nil && TypeOf(d) == IntegerType
}

func FloatP(d *Data) bool {
	return d != nil && TypeOf(d) == FloatType
}

func NumberP(d *Data) bool {
	return IntegerP(d) || FloatP(d)
}

func ObjectP(d *Data) bool {
	return d != nil && TypeOf(d) == BoxedObjectType
}

func FunctionP(d *Data) bool {
	return d != nil && (TypeOf(d) == FunctionType || TypeOf(d) == PrimitiveType)
}

func MacroP(d *Data) bool {
	return d != nil && TypeOf(d) == MacroType
}

func FrameP(d *Data) bool {
	return d != nil && TypeOf(d) == FrameType
}

func Cons(car *Data, cdr *Data) *Data {
	cell := ConsCell{Car: car, Cdr: cdr}
	return &Data{Type: ConsCellType, Value: unsafe.Pointer(&cell)}
}

func AppendBang(l *Data, value *Data) *Data {
	if NilP(l) {
		return Cons(value, nil)
	}

	var c *Data
	for c = l; NotNilP(Cdr(c)); c = Cdr(c) {
	}

	((*ConsCell)(c.Value)).Cdr = Cons(value, nil)

	return l
}

func AppendBangList(l *Data, otherList *Data) *Data {
	if NilP(l) {
		return otherList
	}

	var c *Data
	for c = l; NotNilP(Cdr(c)); c = Cdr(c) {
	}

	((*ConsCell)(c.Value)).Cdr = otherList

	return l
}

func Append(l *Data, value *Data) *Data {
	if NilP(l) {
		return Cons(value, nil)
	}

	var newList = Copy(l)
	var c *Data
	for c = newList; NotNilP(Cdr(c)); c = Cdr(c) {
	}

	((*ConsCell)(c.Value)).Cdr = Cons(value, nil)

	return newList
}

func AppendList(l *Data, otherList *Data) *Data {
	if NilP(l) {
		return otherList
	}

	var newList = Copy(l)
	var c *Data
	for c = newList; NotNilP(Cdr(c)); c = Cdr(c) {
	}

	((*ConsCell)(c.Value)).Cdr = otherList

	return newList
}

func Acons(car *Data, cdr *Data, alist *Data) *Data {
	pair, _ := Assoc(car, alist)
	if NilP(pair) {
		p := ConsCell{Car: car, Cdr: cdr}
		cell := Data{Type: AlistCellType, Value: unsafe.Pointer(&p)}
		conscell := ConsCell{Car: &cell, Cdr: alist}
		return &Data{Type: AlistType, Value: unsafe.Pointer(&conscell)}
	} else {
		((*ConsCell)(pair.Value)).Cdr = cdr
		return alist
	}
}

func Alist(d *Data) *Data {
	if NilP(d) {
		return nil
	}

	if PairP(d) {
		headPair := Car(d)
		return Acons(Car(headPair), Cdr(headPair), Alist(Cdr(d)))
	}

	return d
}

func InternalMakeList(c ...*Data) *Data {
	return ArrayToList(c)
}

func EmptyCons() *Data {
	return Cons(nil, nil)
}

func FrameWithValue(m *FrameMap) *Data {
	return &Data{Type: FrameType, Value: unsafe.Pointer(m)}
}

// func EmptyFrame() *Data {
// 	return &Data{Type: FrameType, Frame: &make(FrameMap)}
// }

func IntegerWithValue(n int64) *Data {
	return &Data{Type: IntegerType, Value: unsafe.Pointer(&n)}
}

func FloatWithValue(n float32) *Data {
	return &Data{Type: FloatType, Value: unsafe.Pointer(&n)}
}

func BooleanWithValue(b bool) *Data {
	if b {
		return LispTrue
	} else {
		return LispFalse
	}
}

func StringWithValue(s string) *Data {
	return &Data{Type: StringType, Value: unsafe.Pointer(&s)}
}

func SymbolWithName(s string) *Data {
	return &Data{Type: SymbolType, Value: unsafe.Pointer(&s)}
}

func NakedSymbolWithName(s string) *Data {
	str := fmt.Sprintf("%s:", s)
	return &Data{Type: SymbolType, Value: unsafe.Pointer(&str)}
}

func NakedSymbolFrom(d *Data) *Data {
	return NakedSymbolWithName(StringValue(d))
}

func FunctionWithNameParamsBodyAndParent(name string, params *Data, body *Data, parentEnv *SymbolTableFrame) *Data {
	return &Data{Type: FunctionType, Value: unsafe.Pointer(MakeFunction(name, params, body, parentEnv))}
}

func MacroWithNameParamsBodyAndParent(name string, params *Data, body *Data, parentEnv *SymbolTableFrame) *Data {
	return &Data{Type: MacroType, Value: unsafe.Pointer(MakeMacro(name, params, body, parentEnv))}
}

func PrimitiveWithNameAndFunc(name string, f *PrimitiveFunction) *Data {
	return &Data{Type: PrimitiveType, Value: unsafe.Pointer(f)}
}

func ObjectWithTypeAndValue(typeName string, o unsafe.Pointer) *Data {
	bo := BoxedObject{ObjType: typeName, Obj: o}
	return &Data{Type: BoxedObjectType, Value: unsafe.Pointer(&bo)}
}

func ConsValue(d *Data) *ConsCell {
	if d == nil {
		return nil
	}

	if PairP(d) || AlistP(d) || DottedPairP(d) {
		return (*ConsCell)(d.Value)
	}

	return nil
}

func Car(d *Data) *Data {
	if d == nil {
		return nil
	}

	if PairP(d) || AlistP(d) || DottedPairP(d) {
		cell := ConsValue(d)
		if cell != nil {
			return cell.Car
		}
	}

	return nil
}

func Cdr(d *Data) *Data {
	if d == nil {
		return nil
	}

	if PairP(d) || AlistP(d) || DottedPairP(d) {
		cell := ConsValue(d)
		if cell != nil {
			return cell.Cdr
		}
	}

	return nil
}

func IntegerValue(d *Data) int64 {
	if d == nil {
		return 0
	}

	if IntegerP(d) {
		return *((*int64)(d.Value))
	}

	if FloatP(d) {
		return int64(*((*float32)(d.Value)))
	}

	return 0
}

func FloatValue(d *Data) float32 {
	if d == nil {
		return 0
	}

	if FloatP(d) {
		return *((*float32)(d.Value))
	}

	if IntegerP(d) {
		return float32(*((*int64)(d.Value)))
	}

	return 0
}

func StringValue(d *Data) string {
	if d == nil {
		return ""
	}

	if StringP(d) || SymbolP(d) {
		return *((*string)(d.Value))
	}

	return ""
}

func BooleanValue(d *Data) bool {
	if d == nil {
		return false
	}

	if BooleanP(d) {
		return ((*BooleanBox)(d.Value)).B
	}

	return true
}

func FrameValue(d *Data) *FrameMap {
	if d == nil {
		return nil
	}

	if FrameP(d) {
		return (*FrameMap)(d.Value)
	}

	return nil
}

func FunctionValue(d *Data) *Function {
	if d == nil {
		return nil
	}

	if d.Type == FunctionType {
		return (*Function)(d.Value)
	}

	return nil
}

func MacroValue(d *Data) *Macro {
	if d == nil {
		return nil
	}

	if d.Type == MacroType {
		return (*Macro)(d.Value)
	}

	return nil
}

func PrimitiveValue(d *Data) *PrimitiveFunction {
	if d == nil {
		return nil
	}

	if d.Type == PrimitiveType {
		return (*PrimitiveFunction)(d.Value)
	}

	return nil
}

func ObjectType(d *Data) (oType string) {
	if d == nil {
		return
	}

	if ObjectP(d) {
		return (*((*BoxedObject)(d.Value))).ObjType
	}

	return
}

func ObjectValue(d *Data) (p unsafe.Pointer) {
	if d == nil {
		return
	}

	if ObjectP(d) {
		return (*((*BoxedObject)(d.Value))).Obj
	}

	return
}

func BoxedObjectValue(d *Data) *BoxedObject {
	if d == nil {
		return nil
	}

	if ObjectP(d) {
		return (*BoxedObject)(d.Value)
	}

	return nil
}

func Length(d *Data) int {
	if d == nil {
		return 0
	}

	if ListP(d) || AlistP(d) {
		return 1 + Length(Cdr(d))
	}

	if FrameP(d) {
		return len(*FrameValue(d))
	}

	return 0
}

func Reverse(d *Data) (result *Data) {
	if d == nil {
		return nil
	}

	if !ListP(d) {
		return d
	}

	var l *Data = nil
	for c := d; NotNilP(c); c = Cdr(c) {
		l = Cons(Car(c), l)
	}

	return l
}

func Flatten(d *Data) (result *Data, err error) {
	if d == nil {
		return nil, nil
	}

	if !ListP(d) {
		return d, nil
	}

	var l []*Data = make([]*Data, 0, 10)
	for c := d; NotNilP(c); c = Cdr(c) {
		if ListP(Car(c)) {
			for i := Car(c); NotNilP(i); i = Cdr(i) {
				l = append(l, Car(i))
			}
		} else {
			l = append(l, Car(c))
		}
	}

	return ArrayToList(l), nil
}

func RecursiveFlatten(d *Data) (result *Data, err error) {
	if d == nil {
		return nil, nil
	}

	if !ListP(d) {
		return d, nil
	}

	var l []*Data = make([]*Data, 0, 10)
	var elem *Data
	for c := d; NotNilP(c); c = Cdr(c) {
		if ListP(Car(c)) {
			elem, err = RecursiveFlatten(Car(c))
			if err != nil {
				return
			}
			for i := elem; NotNilP(i); i = Cdr(i) {
				l = append(l, Car(i))
			}
		} else {
			l = append(l, Car(c))
		}
	}

	return ArrayToList(l), nil
}

func QuoteIt(value *Data) (result *Data) {
	return InternalMakeList(SymbolWithName("quote"), value)
}

func QuoteAll(d *Data) (result *Data) {
	var l []*Data = make([]*Data, 0, 10)
	for c := d; NotNilP(c); c = Cdr(c) {
		l = append(l, QuoteIt(Car(c)))
	}
	return ArrayToList(l)
}

func Assoc(key *Data, alist *Data) (result *Data, err error) {
	for c := alist; NotNilP(c); c = Cdr(c) {
		pair := Car(c)
		if !DottedPairP(pair) && !PairP(pair) {
			err = errors.New("An alist MUST be made of pairs.")
			return
		}
		if IsEqual(Car(pair), key) {
			result = pair
			return
		}
	}
	return
}

func Dissoc(key *Data, alist *Data) (result *Data, err error) {
	var newList *Data = nil
	for c := alist; NotNilP(c); c = Cdr(c) {
		pair := Car(c)
		if !DottedPairP(pair) && !PairP(pair) {
			err = errors.New("An alist MUST be made of pairs.")
			return
		}
		if !IsEqual(Car(pair), key) {
			newList = Acons(Car(pair), Cdr(pair), newList)
		}
	}
	return newList, nil
}

func Copy(d *Data) *Data {
	if d == nil {
		return d
	}

	switch d.Type {
	case AlistType:
		{
			alist := Acons(Copy(Caar(d)), Copy(Cdar(d)), nil)
			for c := Cdr(d); NotNilP(c); c = Cdr(c) {
				alist = Acons(Copy(Caar(c)), Copy(Cdar(c)), alist)
			}
			return alist
		}
	case ConsCellType:
		{
			if NilP(d) {
				return d
			}

			return Cons(Copy(Car(d)), Copy(Cdr(d)))
		}
	case FrameType:
		{
			m := make(FrameMap)
			for k, v := range *(FrameValue(d)) {
				m[k] = Copy(v)
			}
			return FrameWithValue(&m)
		}
	}

	return d
}

func IsEqual(d *Data, o *Data) bool {
	if d == o {
		return true
	}

	if d == nil || o == nil {
		return false
	}

	if AlistP(d) {
		if !AlistP(o) && !ListP(o) {
			return false
		}
	} else if DottedPairP(d) {
		if !PairP(o) && !DottedPairP(o) {
			return false
		}
	} else if TypeOf(o) != TypeOf(d) {
		return false
	}

	if AlistP(d) {
		if Length(d) != Length(o) {
			return false
		}
		for c := d; NotNilP(c); c = Cdr(c) {
			otherPair, err := Assoc(Caar(c), o)
			if err != nil || NilP(otherPair) || !IsEqual(Cdar(c), Cdr(otherPair)) {
				return false
			}
		}
		return true
	}

	if DottedPairP(d) {
		return IsEqual(Car(d), Car(o)) && IsEqual(Cdr(d), Cdr(o))
	}

	if ListP(d) {
		if Length(d) != Length(o) {
			return false
		}
		for a1, a2 := d, o; NotNilP(a1); a1, a2 = Cdr(a1), Cdr(a2) {
			if !IsEqual(Car(a1), Car(a2)) {
				return false
			}
		}
		return true
	}

	if FrameP(d) {
		if len(*(FrameValue(d))) != len(*(FrameValue(o))) {
			return false
		}
		for k, v := range *FrameValue(d) {
			if !IsEqual(v, (*(FrameValue(o)))[k]) {
				return false
			}
		}
		return true
	}

	switch TypeOf(d) {
	case IntegerType:
		return IntegerValue(d) == IntegerValue(o)
	case FloatType:
		return FloatValue(d) == FloatValue(o)
	case BooleanType:
		return BooleanValue(d) == BooleanValue(o)
	case StringType:
		return StringValue(d) == StringValue(o)
	case SymbolType:
		return StringValue(d) == StringValue(o)
	case FunctionType:
		return FunctionValue(d) == FunctionValue(o)
	case MacroType:
		return MacroValue(d) == MacroValue(o)
	case PrimitiveType:
		return PrimitiveValue(d) == PrimitiveValue(o)
	case BoxedObjectType:
		return (ObjectType(d) == ObjectType(o)) && (ObjectValue(d) == ObjectValue(o))
	}

	return false
}

func escapeQuotes(str string) string {
	buffer := make([]rune, 0, 10)
	for _, ch := range str {
		if rune(ch) == '"' {
			buffer = append(buffer, '\\')
		}
		buffer = append(buffer, rune(ch))
	}
	return string(buffer)
}

func String(d *Data) string {
	if d == nil {
		return "()"
	}

	switch d.Type {
	case ConsCellType:
		{
			if NilP(Car(d)) && NilP(Cdr(d)) {
				return "()"
			}
			var c *Data = d

			contents := make([]string, 0, Length(d))
			for NotNilP(c) && PairP(c) {
				contents = append(contents, String(Car(c)))
				c = Cdr(c)
			}
			if c == nil {
				if SymbolP(Car(d)) && StringValue(Car(d)) == "quote" {
					if len(contents) == 1 {
						return fmt.Sprintf("'()")
					} else {
						return fmt.Sprintf("'%s", contents[1])
					}
				} else {
					return fmt.Sprintf("(%s)", strings.Join(contents, " "))
				}
			} else {
				return fmt.Sprintf("(%s . %s)", strings.Join(contents, " "), String(c))
			}
		}
	case AlistType:
		{
			if NilP(Car(d)) && NilP(Cdr(d)) {
				return "()"
			}
			contents := make([]string, 0, Length(d))
			for c := d; NotNilP(c); c = Cdr(c) {
				contents = append(contents, String(Car(c)))
			}
			return fmt.Sprintf("(%s)", strings.Join(contents, " "))
		}
	case AlistCellType:
		return fmt.Sprintf("(%s . %s)", String(Car(d)), String(Cdr(d)))
	case IntegerType:
		return fmt.Sprintf("%d", IntegerValue(d))
	case FloatType:
		{
			raw := fmt.Sprintf("%g", FloatValue(d))
			if strings.ContainsRune(raw, '.') {
				return raw
			}
			return fmt.Sprintf("%s.0", raw)
		}
	case BooleanType:
		if BooleanValue(d) {
			return "#t"
		} else {
			return "#f"
		}
	case StringType:
		return fmt.Sprintf(`"%s"`, escapeQuotes(StringValue(d)))
	case SymbolType:
		return StringValue(d)
	case FunctionType:
		return fmt.Sprintf("<function: %s>", FunctionValue(d).Name)
	case MacroType:
		return fmt.Sprintf("<macro: %s>", MacroValue(d).Name)
	case PrimitiveType:
		return PrimitiveValue(d).String()
	case BoxedObjectType:
		if ObjectType(d) == "[]byte" {
			bytes := (*[]byte)(ObjectValue(d))
			contents := make([]string, 0, len(*bytes))
			for _, b := range *bytes {
				contents = append(contents, fmt.Sprintf("%d", b))
			}
			return fmt.Sprintf("[%s]", strings.Join(contents, " "))
		} else {
			return fmt.Sprintf("<opaque Go object of type %s : 0x%x>", ObjectType(d), (*uint64)(ObjectValue(d)))
		}
	case FrameType:
		pairs := make([]string, 0, len(*FrameValue(d)))
		for key, val := range *FrameValue(d) {
			var valString string
			if FrameP(val) {
				valString = "{...}"
			} else {
				valString = String(val)
			}
			pairs = append(pairs, fmt.Sprintf("%s %s", key, valString))
		}
		return fmt.Sprintf("{%s}", strings.Join(pairs, " "))
	}

	return ""
}

func PrintString(d *Data) string {
	if StringP(d) {
		return StringValue(d)
	} else {
		return String(d)
	}
}

func postProcessFrameShortcuts(d *Data) *Data {
	key := Car(d)
	frame := Cadr(d)
	value := Caddr(d)

	if !SymbolP(key) {
		return d
	}

	s := StringValue(key)
	switch {
	case strings.HasSuffix(s, ":"):
		return InternalMakeList(SymbolWithName("get-slot"), frame, key)
	case strings.HasSuffix(s, ":!"):
		return InternalMakeList(SymbolWithName("set-slot!"), frame, SymbolWithName(strings.TrimSuffix(s, "!")), value)
	case strings.HasSuffix(s, ":?"):
		return InternalMakeList(SymbolWithName("has-slot?"), frame, SymbolWithName(strings.TrimSuffix(s, "?")))
	default:
		return d
	}
}

func printDashes(indent int) {
	for i := indent; i > 0; i -= 1 {
		fmt.Print("-")
	}
}

func logEval(d *Data, env *SymbolTableFrame) {
	if LispTrace && !DebugEvalInDebugRepl {
		depth := env.Depth()
		fmt.Printf("%3d: ", depth)
		printDashes(depth)
		fmt.Printf("> %s\n", String(d))
		EvalDepth += 1
	}
}

func logResult(result *Data, env *SymbolTableFrame) {
	if LispTrace && !DebugEvalInDebugRepl {
		depth := env.Depth()
		fmt.Printf("%3d: <", depth)
		printDashes(depth)
		fmt.Printf(" %s\n", String(result))
	}
}

func evalHelper(d *Data, env *SymbolTableFrame, needFunction bool) (result *Data, err error) {
	if IsInteractive && !DebugEvalInDebugRepl {
		env.CurrentCode.PushFront(fmt.Sprintf("Eval %s", String(d)))
	}

	logEval(d, env)

	if DebugSingleStep {
		DebugSingleStep = false
		DebugRepl(env)
	}

	if DebugCurrentFrame != nil && env == DebugCurrentFrame.Previous {
		DebugCurrentFrame = nil
		DebugRepl(env)
	}

	if d != nil {
		switch d.Type {
		case ConsCellType:
			{
				d = postProcessFrameShortcuts(d)

				var function *Data
				function, err = evalHelper(Car(d), env, true)

				if err != nil {
					return
				}
				if function == nil {
					err = errors.New(fmt.Sprintf("Nil when function or macro expected for %s.", String(Car(d))))
					return
				}

				if !DebugSingleStep && TypeOf(function) == FunctionType && DebugOnEntry.Has(FunctionValue(function).Name) {
					DebugRepl(env)
				}

				args := Cdr(d)

				result, err = Apply(function, args, env)
				if err != nil {
					err = errors.New(fmt.Sprintf("\nEvaling %s. %s", String(d), err))
					return
				} else if DebugReturnValue != nil {
					result = DebugReturnValue
					DebugReturnValue = nil
				}
			}
		case SymbolType:
			if NakedP(d) {
				result = d
			} else {
				result = env.ValueOfWithFunctionSlotCheck(d, needFunction)
			}
		default:
			result = d
		}
	}
	logResult(result, env)
	if IsInteractive && !DebugEvalInDebugRepl {
		env.CurrentCode.Remove(env.CurrentCode.Front())
	}
	return result, nil
}

func Eval(d *Data, env *SymbolTableFrame) (result *Data, err error) {
	return evalHelper(d, env, false)
}

func formatApply(function *Data, args *Data) string {
	var fname string

	if function == nil {
		return "Trying to apply nil!"
	}

	switch function.Type {
	case FunctionType:
		fname = FunctionValue(function).Name
	case MacroType:
		fname = MacroValue(function).Name
	case PrimitiveType:
		fname = PrimitiveValue(function).Name
	}
	return fmt.Sprintf("Apply %s to %s", fname, String(args))
}

func Apply(function *Data, args *Data, env *SymbolTableFrame) (result *Data, err error) {
	if function == nil {
		err = errors.New("Nil when function expected.")
		return
	}
	switch function.Type {
	case FunctionType:
		if env.HasFrame() {
			result, err = FunctionValue(function).ApplyWithFrame(args, env, env.Frame)
		} else {
			result, err = FunctionValue(function).Apply(args, env)
		}
	case MacroType:
		result, err = MacroValue(function).Apply(args, env)
	case PrimitiveType:
		result, err = PrimitiveValue(function).Apply(args, env)
	}

	return
}

func ApplyWithoutEval(function *Data, args *Data, env *SymbolTableFrame) (result *Data, err error) {
	if function == nil {
		err = errors.New("Nil when function or macro expected.")
		return
	}
	switch function.Type {
	case FunctionType:
		result, err = FunctionValue(function).ApplyWithoutEval(args, env)
	case MacroType:
		result, err = MacroValue(function).ApplyWithoutEval(args, env)
	case PrimitiveType:
		result, err = PrimitiveValue(function).ApplyWithoutEval(args, env)
	}

	return
}
