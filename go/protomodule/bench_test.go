package protomodule

import (
	"testing"

	pb "github.com/stripe/skycfg/internal/test_proto"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	benchmarkNewMessage_empty_sink *protoMessage
	benchmarkNewMessage_empty_err  error
)

func BenchmarkNewMessage_empty(b *testing.B) {
	wantMsg := &pb.MessageV2{}
	benchmarkNewMessage_empty_sink, benchmarkNewMessage_empty_err = NewMessage(wantMsg)
	b.ResetTimer()
	for range b.N {
		benchmarkNewMessage_empty_sink, benchmarkNewMessage_empty_err = NewMessage(wantMsg)
	}
}

var (
	benchmarkNewMessage_full_sink *protoMessage
	benchmarkNewMessage_full_err  error
)

func BenchmarkNewMessage_full(b *testing.B) {
	wantMsg := &pb.MessageV2{
		FInt32:   proto.Int32(1010),
		FInt64:   proto.Int64(1020),
		FUint32:  proto.Uint32(1030),
		FUint64:  proto.Uint64(1040),
		FFloat32: proto.Float32(10.50),
		FFloat64: proto.Float64(10.60),
		FString:  proto.String("some string"),
		FBool:    proto.Bool(true),
		FSubmsg: &pb.MessageV2{
			FString: proto.String("string in submsg"),
		},
		RString: []string{"r_string1", "r_string2"},
		RSubmsg: []*pb.MessageV2{{
			FString: proto.String("string in r_submsg"),
		}},
		MapString: map[string]string{
			"map_string key": "map_string val",
		},
		MapSubmsg: map[string]*pb.MessageV2{
			"map_submsg key": {
				FString: proto.String("map_submsg val"),
			},
		},
		FNestedSubmsg: &pb.MessageV2_NestedMessage{
			FString: proto.String("nested_submsg val"),
		},
		FToplevelEnum: pb.ToplevelEnumV2_TOPLEVEL_ENUM_V2_B.Enum(),
		FNestedEnum:   pb.MessageV2_NESTED_ENUM_B.Enum(),
		FOneof:        &pb.MessageV2_FOneofB{FOneofB: "string in oneof"},
		FBytes:        []byte("also some string"),
		F_BoolValue:   &wrapperspb.BoolValue{Value: true},
		F_StringValue: &wrapperspb.StringValue{Value: "something"},
		F_DoubleValue: &wrapperspb.DoubleValue{Value: 3110.4120},
		F_Int32Value:  &wrapperspb.Int32Value{Value: 110},
		F_Int64Value:  &wrapperspb.Int64Value{Value: 2148483647},
		F_BytesValue:  &wrapperspb.BytesValue{Value: []byte("foo/bar/baz")},
		F_Uint32Value: &wrapperspb.UInt32Value{Value: 4294967295},
		F_Uint64Value: &wrapperspb.UInt64Value{Value: 8294967295},
		R_StringValue: []*wrapperspb.StringValue{
			{Value: "s1"},
			{Value: "s2"},
			{Value: "s3"},
		},
	}
	benchmarkNewMessage_full_sink, benchmarkNewMessage_full_err = NewMessage(wantMsg)

	b.ResetTimer()
	for range b.N {
		benchmarkNewMessage_full_sink, benchmarkNewMessage_full_err = NewMessage(wantMsg)
	}
}

var (
	benchmarkMessage_Attr_unpopulatedScalar_sink starlark.Value
	benchmarkMessage_Attr_unpopulatedScalar_err  error
)

func BenchmarkMessage_Attr_unpopulatedScalar(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		benchmarkMessage_Attr_unpopulatedScalar_sink, benchmarkMessage_Attr_unpopulatedScalar_err = msg.Attr("f_int32")
	}
}

var (
	benchmarkMessage_Attr_populatedScalar_sink starlark.Value
	benchmarkMessage_Attr_populatedScalar_err  error
)

func BenchmarkMessage_Attr_populatedScalar(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{
		FInt32: 42,
	})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for range b.N {
		benchmarkMessage_Attr_populatedScalar_sink, benchmarkMessage_Attr_populatedScalar_err = msg.Attr("f_int32")
	}
}

var (
	benchmarkMessage_Attr_unpopulatedList_sink starlark.Value
	benchmarkMessage_Attr_unpopulatedList_err  error
)

func BenchmarkMessage_Attr_unpopulatedList(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{})
	if err != nil {
		b.Fatal(err)
	}
	msg.Freeze()
	b.ResetTimer()
	for range b.N {
		benchmarkMessage_Attr_unpopulatedList_sink, benchmarkMessage_Attr_unpopulatedList_err = msg.Attr("r_string")
	}
}

var (
	benchmarkMessage_Attr_populatedList_sink starlark.Value
	benchmarkMessage_Attr_populatedList_err  error
)

func BenchmarkMessage_Attr_populatedList(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{
		RString: []string{"a", "b", "c"},
	})
	if err != nil {
		b.Fatal(err)
	}
	msg.Freeze()
	b.ResetTimer()
	for range b.N {
		benchmarkMessage_Attr_populatedList_sink, benchmarkMessage_Attr_populatedList_err = msg.Attr("r_string")
	}
}

var (
	benchmarkMessage_Attr_unpopulatedLastItem_sink starlark.Value
	benchmarkMessage_Attr_unpopulatedLastItem_err  error
)

func BenchmarkMessage_Attr_unpopulatedLastItem(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for range b.N {
		benchmarkMessage_Attr_unpopulatedLastItem_sink, benchmarkMessage_Attr_unpopulatedLastItem_err = msg.Attr("safe_")
	}
}

var (
	benchmarkMessage_Attr_populatedLastItem_sink starlark.Value
	benchmarkMessage_Attr_populatedLastItem_err  error
)

func BenchmarkMessage_Attr_populatedLastItem(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{Safe_: true})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for range b.N {
		benchmarkMessage_Attr_populatedLastItem_sink, benchmarkMessage_Attr_populatedLastItem_err = msg.Attr("safe_")
	}
}

var (
	benchmarkMessage_SetField_unpopulatedInt32_err error
)

func BenchmarkMessage_SetField_unpopulatedInt32(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	si := starlark.MakeInt(100)
	for range b.N {
		benchmarkMessage_SetField_unpopulatedInt32_err = msg.SetField("f_int32", si)
	}
}

var (
	benchmarkMessage_SetField_populatedInt32_err error
)

func BenchmarkMessage_SetField_populatedInt32(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{
		FInt32: 42,
	})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	si := starlark.MakeInt(100)
	for range b.N {
		benchmarkMessage_SetField_populatedInt32_err = msg.SetField("f_int32", si)
	}
}

var (
	benchmarkMessage_SetField_unpopulatedLastItem_err error
)

func BenchmarkMessage_SetField_unpopulatedLastItem(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for range b.N {
		benchmarkMessage_SetField_unpopulatedLastItem_err = msg.SetField("safe_", starlark.True)
	}
}

var (
	benchmarkMessage_SetField_populatedLastItem_err error
)

func BenchmarkMessage_SetField_populatedLastItem(b *testing.B) {
	msg, err := NewMessage(&pb.MessageV3{Safe_: true})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for range b.N {
		benchmarkMessage_SetField_populatedLastItem_err = msg.SetField("safe_", starlark.True)
	}
}
