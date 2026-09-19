package controller

import (
	"net/http"

	"rain-yi-backend/model"
	"rain-yi-backend/utils"
)

func absUserJSON(r *http.Request, u *model.User) map[string]any {
	if u == nil {
		return nil
	}
	return map[string]any{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
		"avatar":   utils.AssetURL(r, u.Avatar),
	}
}

func absPersona(r *http.Request, p *model.Persona) model.Persona {
	if p == nil {
		return model.Persona{}
	}
	out := *p
	out.Avatar = utils.AssetURL(r, out.Avatar)
	return out
}

func absConversation(r *http.Request, conv *model.Conversation) model.Conversation {
	if conv == nil {
		return model.Conversation{}
	}
	out := *conv
	out.AIAvatar = utils.AssetURL(r, out.AIAvatar)
	return out
}

func absMessages(r *http.Request, msgs []model.Message) []model.Message {
	if msgs == nil {
		return nil
	}
	out := make([]model.Message, len(msgs))
	for i, m := range msgs {
		m.AudioURL = utils.AssetURL(r, m.AudioURL)
		m.AttachmentURL = utils.AssetURL(r, m.AttachmentURL)
		out[i] = m
	}
	return out
}

func absFileRecord(r *http.Request, rec *model.FileRecord) *model.FileRecord {
	if rec == nil {
		return nil
	}
	out := *rec
	out.URL = utils.AssetURL(r, out.URL)
	return &out
}

func absVoice(r *http.Request, v *model.UserVoice) *model.UserVoice {
	if v == nil {
		return nil
	}
	out := *v
	out.URL = utils.AssetURL(r, out.URL)
	return &out
}
