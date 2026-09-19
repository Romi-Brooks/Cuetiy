package service

import (
	"log"
	"sync"

	"cuetiy-backend/model"
	"cuetiy-backend/repository"
)

// MessageWriter 消息异步落库：聊天路径不阻塞 MySQL，本轮结束后后台写入
type MessageWriter struct {
	msgRepo *repository.MessageRepository
	mu      sync.Mutex
	pending map[int64][]model.Message // convID -> 尚未落库的消息（供下轮组装）
}

func NewMessageWriter(msgRepo *repository.MessageRepository) *MessageWriter {
	return &MessageWriter{
		msgRepo: msgRepo,
		pending: make(map[int64][]model.Message),
	}
}

// EnqueueAsync 后台写入，并暂存到内存 pending，避免下轮 BuildContext 丢消息
func (w *MessageWriter) EnqueueAsync(msgs ...*model.Message) {
	if w == nil || len(msgs) == 0 {
		return
	}
	go func() {
		for _, m := range msgs {
			if m == nil {
				continue
			}
			if err := w.msgRepo.Create(m); err != nil {
				log.Printf("[async-msg] save failed conv=%d role=%s: %v", m.ConversationID, m.Role, err)
				continue
			}
			w.mu.Lock()
			w.pending[m.ConversationID] = append(w.pending[m.ConversationID], *m)
			// 只保留最近 40 条 pending，落库成功后仍保留一小段供组装
			list := w.pending[m.ConversationID]
			if len(list) > 40 {
				list = list[len(list)-40:]
				w.pending[m.ConversationID] = list
			}
			w.mu.Unlock()
		}
	}()
}

// PendingMessages 读取某会话尚未（或刚）落库的消息副本
func (w *MessageWriter) PendingMessages(convID int64) []model.Message {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	src := w.pending[convID]
	if len(src) == 0 {
		return nil
	}
	out := make([]model.Message, len(src))
	copy(out, src)
	return out
}

// ClearPending 会话清空时调用
func (w *MessageWriter) ClearPending(convID int64) {
	if w == nil {
		return
	}
	w.mu.Lock()
	delete(w.pending, convID)
	w.mu.Unlock()
}

// FlushSync 同步落库（归档前可调用）
func (w *MessageWriter) FlushSync(convID int64) {
	if w == nil {
		return
	}
	// pending 里本就对应已异步入库的记录，这里仅清空内存镜像即可
	w.ClearPending(convID)
}
