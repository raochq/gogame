package errcode

//go:generate stringer -type ErrCode -linecomment -output code_string.go
type ErrCode int32

const (
	EC_OK               = ErrCode(0)   //OK
	EC_Unknown          = ErrCode(-1)  //未知错误
	EC_FormatError      = ErrCode(-2)  //数据格式错误
	EC_DataNotExists    = ErrCode(-3)  //数据不存在
	EC_MarshalFail      = ErrCode(-4)  //协议编码错误
	EC_UnmarshalFail    = ErrCode(-5)  //协议解码错误
	EC_NetworkFail      = ErrCode(-6)  //网络错误
	EC_ParameterInvalid = ErrCode(-7)  //参数错误
	EC_UnknownMessage   = ErrCode(-8)  //无法识别的消息
	EC_GamesvrError     = ErrCode(-9)  //游戏服务器链接错误
	EC_TokenInvalid     = ErrCode(-10) //Token无效(验证未通过)
	EC_AccountReplaced  = ErrCode(-11) //账号被新登录踢出
	EC_ServerBusy       = ErrCode(-12) //服务器忙，请稍后再试
)

func (i ErrCode) Error() string {
	return i.String()
}
