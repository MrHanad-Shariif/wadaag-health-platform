import { apiClient } from '../../api/client'
import type { Conversation, CreateConversationInput, Message, PostMessageInput } from './types'

export function listConversations() {
  return apiClient.get<Conversation[]>('/api/v1/conversations')
}

export function createConversation(otherUserId: string) {
  const input: CreateConversationInput = { other_user_id: otherUserId }
  return apiClient.post<Conversation>('/api/v1/conversations', input)
}

export function listMessages(conversationId: string) {
  return apiClient.get<Message[]>(`/api/v1/conversations/${conversationId}/messages`)
}

export function sendMessage(conversationId: string, body: string) {
  const input: PostMessageInput = { body }
  return apiClient.post<Message>(`/api/v1/conversations/${conversationId}/messages`, input)
}

export function markRead(conversationId: string) {
  return apiClient.post<void>(`/api/v1/conversations/${conversationId}/read`, undefined)
}
