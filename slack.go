package main

import (
	"encoding/json"
)

type SelectReply struct {
	MainText      string
	IsInChannel   bool
	SecondaryText string
	CallbackId    string
	ActionName    string
	DropDownText  string
	Options       []Option
}

type Option struct {
	Text  string `json:"text"`
	Value string `json:"value"`
}

func (reply SelectReply) FormatReply() []byte {
	var responseType string
	if reply.IsInChannel {
		responseType = "in_channel"
	} else {
		responseType = "ephemeral"
	}
	formatted, err := json.Marshal(map[string]interface{}{
		"text":          reply.MainText,
		"response_type": responseType,
		"attachments": []interface{}{
			map[string]interface{}{
				"text":            reply.SecondaryText,
				"attachment_type": "default",
				"callback_id":     reply.CallbackId,
				"actions": []interface{}{
					map[string]interface{}{
						"name":    reply.ActionName,
						"text":    reply.DropDownText,
						"type":    "select",
						"options": reply.Options,
					},
				},
			},
		},
	})

	if err != nil {
		return []byte{}
	}
	return formatted
}
