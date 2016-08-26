package pistis

func (m *Message)asOpenCloseMsg(onFailed func())(bool,string,string,string){
	msg := m.Payload.(map[string]interface {})
	ok:=msg["Token"]!=nil && msg["Username"]!=nil && msg["UUID"]!=nil
	if !ok {
		if onFailed != nil {
			onFailed()
		}
		return ok,"","",""
	}
		return ok,msg["Token"].(string),msg["Username"].(string),msg["UUID"].(string)
}
