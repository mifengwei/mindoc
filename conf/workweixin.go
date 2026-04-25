package conf

type WorkWeixinConf struct {
	CorpId  string // 企业ID
	AgentId string // 应用ID
	Secret  string // 应用密钥
	// ContactSecret string // 通讯录密钥
}

func GetWorkWeixinConfig() *WorkWeixinConf {
	corpid, _ := GetString("workweixin_corpid")
	agentid, _ := GetString("workweixin_agentid")
	secret, _ := GetString("workweixin_secret")
	// contact_secret, _ := GetString("workweixin_contact_secret")

	c := &WorkWeixinConf{
		CorpId:  corpid,
		AgentId: agentid,
		Secret:  secret,
		// ContactSecret: contact_secret,
	}
	return c
}
