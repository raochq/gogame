package errcode

//go:generate stringer -type ErrCode -linecomment -output code_string.go
type ErrCode int32

const (
	EC_OK                    = ErrCode(0)  //OK
	EC_Unknown               = ErrCode(-1) //未知错误
	EC_FormatError           = ErrCode(-2) //数据格式错误
	EC_DataNotExists         = ErrCode(-3) //数据不存在
	EC_ProtobufMarshalFail   = ErrCode(-4) //Pb编码错误
	EC_ProtobufUnmarshalFail = ErrCode(-5) //Pb解码错误
	EC_NetworkFail           = ErrCode(-6) //网络错误
	EC_ParameterInvalid      = ErrCode(-7) //参数错误
	EC_UnknownMessage        = ErrCode(-8) //无法识别的消息
)

func (e ErrCode) Error() string {
	return e.String()
}
