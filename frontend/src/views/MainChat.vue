<template>
  <div class="h-screen flex overflow-hidden bg-white dark:bg-wechat-bg-dark">
    <!-- Desktop: Left Tab Bar -->
    <div v-if="!isMobile" class="flex flex-col items-center w-16 bg-gray-50 dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 py-4 shrink-0">
      <button v-for="t in tabs" :key="t.key"
        class="w-12 h-12 rounded-xl flex flex-col items-center justify-center gap-0.5 transition-colors"
        :class="activeTab === t.key ? 'bg-wechat-green/10 text-wechat-green' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800'"
        @click="activeTab = t.key"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" v-html="t.icon" />
        <span class="text-[10px] font-medium">{{ t.label }}</span>
      </button>
      <div class="flex-1" />
      <button class="w-12 h-12 rounded-xl flex flex-col items-center justify-center text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
        @click="handleLogout"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/>
        </svg>
        <span class="text-[10px] font-medium">退出</span>
      </button>
    </div>

    <!-- ==================== CHAT TAB ==================== -->
    <template v-if="activeTab === 'chat'">
      <div v-if="isMobile && showSidebar" class="fixed inset-0 bg-black bg-opacity-50 z-10 lg:hidden" @click="showSidebar = false" />

      <aside class="flex flex-col bg-white dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 flex-shrink-0 z-20"
        :class="isMobile ? (showSidebar ? 'fixed inset-y-0 left-0 w-full' : 'hidden') : 'w-80'"
      >
        <div class="flex items-center justify-between px-5 py-4 border-b border-gray-200 dark:border-gray-700">
          <h1 class="text-xl font-bold text-gray-800 dark:text-gray-100">{{ userStore.username }}</h1>
        </div>

        <div class="px-4 py-3">
          <div class="relative">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input v-model="searchQuery" type="text" placeholder="搜索会话" class="w-full pl-10 pr-4 py-2 text-sm bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-xl border-none outline-none focus:ring-2 focus:ring-wechat-green/30 transition-all" />
          </div>
        </div>

        <div class="flex-1 overflow-y-auto">
          <div v-for="conv in filteredConversations" :key="conv.id"
            class="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors border-b border-gray-100 dark:border-gray-800"
            :class="conv.id === chatStore.currentConversationId ? 'bg-gray-100 dark:bg-gray-800' : ''"
            @click="selectConversation(conv)"
          >
            <div class="flex-shrink-0 w-12 h-12 rounded-full overflow-hidden">
              <img v-if="conv.ai_avatar" :src="conv.ai_avatar" alt="avatar" class="w-full h-full object-cover" @error="(conv as any).ai_avatar = ''" />
              <div v-else class="w-full h-full bg-wechat-green bg-opacity-10 flex items-center justify-center text-wechat-green text-lg font-medium">{{ (conv.title || 'R').charAt(0) }}</div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between">
                <h3 class="text-sm font-medium text-gray-800 dark:text-gray-200 truncate">{{ conv.ai_nickname || conv.title || '会话' }}</h3>
                <span class="text-xs text-gray-400 flex-shrink-0 ml-2">{{ formatConversationTime(conv.updated_at) }}</span>
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1 truncate">{{ conv.last_message?.content ? truncate(conv.last_message.content, 40) : '暂无消息' }}</p>
            </div>
          </div>

          <div v-if="filteredConversations.length === 0" class="flex flex-col items-center justify-center py-16 text-gray-400 dark:text-gray-500">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-12 h-12 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            <p class="text-sm">{{ searchQuery ? '没有找到匹配的会话' : '暂无会话，开始一段新对话吧' }}</p>
          </div>
        </div>

        <div class="px-4 py-3 border-t border-gray-200 dark:border-gray-700">
          <button class="w-full flex items-center justify-center gap-2 py-2.5 text-sm font-medium text-wechat-green hover:bg-wechat-green/5 rounded-xl transition-colors border border-wechat-green/30" @click="showNewConvModal = true">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
            新建会话
          </button>
        </div>
      </aside>

      <main class="flex-1 flex flex-col min-w-0" :class="isMobile && showSidebar ? 'hidden' : ''">
        <template v-if="chatStore.currentConversationId">
          <!-- Chat Header -->
          <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
            <div class="flex items-center gap-3">
              <button v-if="isMobile" class="p-1 rounded-lg text-gray-500 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors" @click="showSidebar = true">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>
              </button>
              <div class="flex items-center gap-3">
                <div class="w-9 h-9 rounded-full overflow-hidden flex-shrink-0">
                  <img v-if="!aiAvatarError" :src="chatStore.aiAvatar" :alt="chatStore.aiNickname" class="w-full h-full object-cover" @error="aiAvatarError = true" />
                  <div v-else class="w-full h-full bg-purple-500 flex items-center justify-center text-white text-sm font-medium">{{ chatStore.aiNickname.charAt(0) }}</div>
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <h2 class="text-base font-semibold text-gray-800 dark:text-gray-100">{{ chatStore.aiNickname }}</h2>
                    <span class="text-xs px-2 py-0.5 rounded-full bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400">{{ chatStore.currentPersonaName }}</span>
                  </div>
                  <p class="text-xs text-gray-400">{{ chatStore.isStreaming ? '对方正在输入...' : chatStore.wsConnected ? '在线' : '连接中...' }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- Messages -->
          <div ref="messagesContainer" class="flex-1 overflow-y-auto px-4 py-4 space-y-1 bg-gray-50 dark:bg-gray-800/50">
            <div ref="loadMoreSentinel" class="h-4 flex items-center justify-center"><span v-if="chatStore.isLoading" class="text-xs text-gray-400">加载中...</span></div>
            <template v-for="(msg, idx) in chatStore.messages" :key="idx">
              <TimeStamp v-if="shouldShowTimestamp(msg, idx)" :time="msg.created_at" />
              <ChatBubble :message="msg" :isAI="msg.role === 'assistant'" :aiNickname="chatStore.aiNickname" :aiAvatar="chatStore.aiAvatar" :userNickname="userStore.username" :userAvatar="userStore.userAvatar" />
              <div v-if="msg.role === 'assistant'" class="flex justify-start pl-14 -mt-2 mb-3">
                <span class="text-[10px] text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 cursor-pointer transition-colors"
                  @click="toggleMsgDebug(msg.id, idx)"
                >{{ debugExpandedMsgId === msg.id ? '收起 Debug' : 'Debug' }}</span>
              </div>
              <div v-if="debugExpandedMsgId === msg.id && debugMsgContent" class="bg-gray-900 text-gray-100 text-xs rounded-xl px-4 py-3 mb-2 mx-14 whitespace-pre-wrap break-all max-h-48 overflow-y-auto">{{ debugMsgContent }}</div>
            </template>
            <div ref="bottomSentinel" />
          </div>

          <!-- Input -->
          <div class="bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 px-4 py-3">
            <div class="flex items-end gap-3">
              <div class="flex items-center gap-1 pb-1">
                <button class="flex items-center justify-center w-9 h-9 rounded-lg text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors" @click="showEmojiPicker = true">
                  <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/></svg>
                </button>
                <VoiceButton />
              </div>
              <div class="flex-1 min-w-0">
                <textarea ref="textareaRef" v-model="inputContent" class="w-full resize-none bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200 rounded-xl px-4 py-2.5 text-sm outline-none focus:ring-2 focus:ring-wechat-green/30 transition-all leading-relaxed" rows="1" placeholder="输入消息..." :disabled="chatStore.isStreaming" @input="autoGrow" @keydown="onKeydown" />
              </div>
              <button class="flex-shrink-0 flex items-center justify-center w-10 h-10 rounded-xl transition-colors" :class="inputContent.trim() && !chatStore.isStreaming ? 'bg-wechat-green hover:bg-wechat-green-dark text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-400 dark:text-gray-500 cursor-not-allowed'" :disabled="!inputContent.trim() || chatStore.isStreaming" @click="sendMessage">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>
              </button>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="flex-1 flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-800/50 text-gray-400 dark:text-gray-500">
            <div class="w-24 h-24 rounded-full bg-wechat-green/10 flex items-center justify-center mb-6">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-12 h-12 text-wechat-green" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            </div>
            <h3 class="text-lg font-medium text-gray-600 dark:text-gray-400 mb-2">欢迎使用 RainYi</h3>
            <p class="text-sm text-center max-w-xs">选择或创建一个会话，开始你的情感陪伴之旅</p>
          </div>
        </template>
      </main>
    </template>

    <!-- ==================== PERSONA TAB ==================== -->
    <template v-if="activeTab === 'persona'">
      <div class="flex-1 flex flex-col bg-white dark:bg-gray-900 overflow-hidden">
        <!-- Header: Search + Add button -->
        <div class="px-5 py-4 border-b border-gray-200 dark:border-gray-700 space-y-3">
          <div class="flex items-center gap-2">
            <div class="relative flex-1">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
              <input v-model="personaSearchQuery" type="text" placeholder="搜索人格" class="w-full pl-10 pr-4 py-2 text-sm bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded-xl border-none outline-none focus:ring-2 focus:ring-wechat-green/30 transition-all" />
            </div>
            <button class="shrink-0 flex items-center gap-1.5 px-3 py-2 text-sm font-medium text-white bg-wechat-green hover:bg-wechat-green-dark rounded-xl transition-colors" @click="openAddPersonaModal">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              添加
            </button>
          </div>
          <p class="text-xs text-gray-400">{{ filteredPersonas.length }} 个角色</p>
        </div>

        <!-- Persona List -->
        <div class="flex-1 overflow-y-auto">
          <div v-for="p in filteredPersonas" :key="p.id"
            class="flex items-center gap-3 px-5 py-3.5 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors border-b border-gray-100 dark:border-gray-800/50"
            @click="openPersonaSettings(p)"
          >
            <div class="w-12 h-12 rounded-full shrink-0 overflow-hidden" :style="{ backgroundColor: personaColor(p.name) }">
              <img v-if="p.avatar" :src="p.avatar" class="w-full h-full object-cover" @error="(p as any).avatar = ''" />
              <div v-else class="w-full h-full flex items-center justify-center text-white text-base font-bold">{{ p.name.charAt(0).toUpperCase() }}</div>
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="text-sm font-semibold text-gray-800 dark:text-gray-200">{{ p.nickname || p.name }}</p>
                <span v-if="p.is_built_in" class="text-[10px] text-wechat-green bg-wechat-green/10 rounded-full px-1.5 py-0.5">官方</span>
              </div>
              <p class="text-xs text-gray-400 truncate mt-0.5">{{ p.description || '这个人很懒，什么都没留下~' }}</p>
            </div>
            <button class="text-xs text-wechat-green border border-wechat-green/30 rounded-lg px-3 py-1 hover:bg-wechat-green/5 transition-colors shrink-0" @click.stop="switchToPersona(p.id)">对话</button>
          </div>
          <div v-if="filteredPersonas.length === 0" class="flex flex-col items-center justify-center py-24 text-gray-400">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-16 h-16 mb-4 text-gray-300 dark:text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
            <p class="text-sm">{{ personaSearchQuery ? '没有匹配的角色' : '暂无可用角色' }}</p>
          </div>
        </div>
      </div>
    </template>

    <!-- ==================== MY TAB ==================== -->
    <template v-if="activeTab === 'my'">
      <div class="flex-1 flex flex-col bg-gray-50 dark:bg-gray-900 overflow-hidden">
        <div class="flex-1 overflow-y-auto">
          <div class="max-w-2xl mx-auto px-4 py-4 space-y-4">
            <!-- Toast -->
            <Transition name="fade">
              <div v-if="showSuccess" class="fixed top-4 left-1/2 -translate-x-1/2 z-50 bg-green-500 text-white text-sm px-4 py-2 rounded-full shadow-lg">{{ successMessage }}</div>
            </Transition>

            <!-- User Info -->
            <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm overflow-hidden">
              <div class="px-5 py-4 border-b border-gray-100 dark:border-gray-700">
                <h2 class="text-base font-semibold text-gray-800 dark:text-gray-100">用户信息</h2>
              </div>
              <div class="px-5 py-4 space-y-3">
                <div class="flex items-center justify-between">
                  <span class="text-sm text-gray-500 dark:text-gray-400">用户名</span>
                  <span class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ userStore.username }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span class="text-sm text-gray-500 dark:text-gray-400">邮箱</span>
                  <span class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ userStore.userInfo?.email || '未设置' }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span class="text-sm text-gray-500 dark:text-gray-400">头像</span>
                  <div class="flex items-center gap-2">
                    <img :src="userStore.userAvatar" class="w-8 h-8 rounded-full object-cover border border-gray-200 dark:border-gray-600" @error="(userStore as any).userInfo = { ...(userStore.userInfo || {}), avatar: '' }" />
                    <label class="text-xs text-wechat-green cursor-pointer hover:text-wechat-green-dark transition-colors flex items-center gap-1">
                      <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                      <span>{{ uploadingAvatarFor === 'user' ? '上传中...' : '更换' }}</span>
                      <input type="file" accept="image/*" class="hidden" :disabled="uploadingAvatarFor !== null" @change="uploadUserAvatar($event)" />
                    </label>
                  </div>
                </div>
                <div class="pt-2">
                  <button class="w-full text-sm text-red-500 border border-red-300 dark:border-red-700 rounded-lg py-2.5 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors font-medium" @click="handleLogout">退出登录</button>
                </div>
              </div>
            </div>
            <div class="h-6" />
          </div>
        </div>
      </div>
    </template>

    <!-- ==================== MOBILE BOTTOM TAB BAR ==================== -->
    <div v-if="isMobile && !(activeTab === 'chat' && chatStore.currentConversationId)" class="fixed bottom-0 left-0 right-0 z-30 flex bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 pb-[env(safe-area-inset-bottom,0px)]">
      <button v-for="t in tabs" :key="t.key" class="flex-1 flex flex-col items-center justify-center py-2 gap-0.5 transition-colors"
        :class="activeTab === t.key ? 'text-wechat-green' : 'text-gray-500 dark:text-gray-400'"
        @click="activeTab = t.key"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" v-html="t.icon" />
        <span class="text-[10px] font-medium">{{ t.label }}</span>
      </button>
    </div>

    <!-- Add Persona Modal -->
    <Transition name="fade">
      <div v-if="showAddPersonaModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" @click.self="closeAddPersonaModal">
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-full max-w-lg mx-4 overflow-hidden" @click.stop>
          <div class="px-5 py-4 border-b border-gray-100 dark:border-gray-700">
            <h3 class="text-base font-semibold text-gray-800 dark:text-gray-100">添加人格</h3>
            <p class="text-xs text-gray-400 mt-0.5">设置名称并上传 .md 技能文件</p>
          </div>
          <div class="px-5 py-4 space-y-4">
              <div>
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400">人格名（英文）</label>
                <input v-model="addPersonaSystemName" class="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200" placeholder="例：rain（用于系统标识）" />
              </div>
              <div>
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400">昵称（显示名称）</label>
                <input v-model="addPersonaNickname" class="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200" placeholder="例：Rain（显示在聊天和人格列表）" />
              </div>
            <div>
              <label class="text-xs font-medium text-gray-500 dark:text-gray-400">技能文件 (.md)</label>
              <label class="mt-1 flex flex-col items-center justify-center border-2 border-dashed border-gray-200 dark:border-gray-600 rounded-xl px-4 py-6 cursor-pointer hover:border-wechat-green/50 transition-colors">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8 text-gray-300 dark:text-gray-500 mb-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                <span class="text-sm text-gray-400">点击选择文件，支持多选</span>
                <span class="text-xs text-gray-400 mt-1">每个文件不超过 10KB</span>
                <input type="file" accept=".md" multiple class="hidden" @change="onAddPersonaFilesChange" />
              </label>
            </div>
            <div v-if="addPersonaFiles.length > 0" class="space-y-1.5">
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">已选 {{ addPersonaFiles.length }} 个文件</p>
              <div v-for="(f, i) in addPersonaFiles" :key="f.name" class="flex items-center justify-between bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2">
                <div class="flex-1 min-w-0">
                  <p class="text-xs font-medium text-gray-700 dark:text-gray-300 truncate">{{ f.name }}</p>
                  <p class="text-xs text-gray-400">优先级 {{ i }} · {{ (f.size / 1024).toFixed(1) }} KB</p>
                </div>
                <button class="text-xs text-red-400 hover:text-red-600 shrink-0 ml-2" @click="removeAddPersonaFile(i)">移除</button>
              </div>
              <p v-if="addPersonaFileErrors.length" v-for="err in addPersonaFileErrors" :key="err" class="text-xs text-red-400">{{ err }}</p>
            </div>
          </div>
          <div class="px-5 py-4 border-t border-gray-100 dark:border-gray-700 flex justify-end gap-2">
            <button class="text-sm text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-750 rounded-lg px-4 py-2 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors" @click="closeAddPersonaModal">取消</button>
            <button class="text-sm text-white bg-wechat-green hover:bg-wechat-green-dark rounded-lg px-4 py-2 font-medium transition-colors disabled:opacity-50"
              :disabled="!addPersonaSystemName.trim() || !addPersonaNickname.trim() || addPersonaFiles.length === 0 || submittingAddPersona"
              @click="submitAddPersona">{{ submittingAddPersona ? '提交中...' : '创建' }}</button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Edit Persona Modal -->
    <Transition name="fade">
      <div v-if="editingPersona" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" @click.self="editingPersona = null">
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-full max-w-lg mx-4 overflow-hidden" @click.stop>
          <div class="px-5 py-4 border-b border-gray-100 dark:border-gray-700">
            <h3 class="text-base font-semibold text-gray-800 dark:text-gray-100">人格设置</h3>
            <p class="text-xs text-gray-400 mt-0.5">编辑 {{ editingPersona.nickname || editingPersona.name }}</p>
          </div>
          <div class="px-5 py-4 space-y-4 max-h-[60vh] overflow-y-auto">
            <!-- Avatar -->
            <div class="flex items-center gap-4">
              <div class="w-16 h-16 rounded-full flex items-center justify-center text-white text-xl font-bold shrink-0" :style="{ backgroundColor: personaColor(editingPersona.name) }">
                <img v-if="editPersonaAvatarUrl" :src="editPersonaAvatarUrl" class="w-full h-full rounded-full object-cover" />
                <span v-else>{{ editingPersona.name.charAt(0).toUpperCase() }}</span>
              </div>
              <label class="text-xs text-wechat-green cursor-pointer hover:text-wechat-green-dark transition-colors flex items-center gap-1 border border-wechat-green/30 rounded-lg px-3 py-1.5">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                <span>{{ uploadingPersonaAvatar ? '上传中...' : '更换头像' }}</span>
                <input type="file" accept="image/*" class="hidden" :disabled="uploadingPersonaAvatar" @change="uploadPersonaAvatar($event)" />
              </label>
            </div>

            <!-- Name & Description -->
            <div>
              <label class="text-xs font-medium text-gray-500 dark:text-gray-400">人格名</label>
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-1 px-1">{{ editingPersona.name }}</p>
            </div>
            <div>
              <label class="text-xs font-medium text-gray-500 dark:text-gray-400">昵称（显示名称）</label>
              <input v-model="editPersonaName" class="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200" />
            </div>
            <div>
              <label class="text-xs font-medium text-gray-500 dark:text-gray-400">描述</label>
              <input v-model="editPersonaDesc" class="w-full text-sm border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2 mt-1 outline-none focus:border-wechat-green dark:text-gray-200" placeholder="个性签名" />
            </div>

            <!-- Skill Files -->
            <div>
              <div class="flex items-center justify-between mb-2">
                <label class="text-xs font-medium text-gray-500 dark:text-gray-400">技能文件</label>
                <label class="text-xs text-wechat-green cursor-pointer hover:text-wechat-green-dark flex items-center gap-0.5">
                  <svg xmlns="http://www.w3.org/2000/svg" class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                  <span>上传</span>
                  <input type="file" accept=".md" multiple class="hidden" @change="uploadPersonaSkillFiles($event)" />
                </label>
              </div>
              <div v-if="editingPersona && personaFilesMap[editingPersona.id]?.length" class="space-y-1.5">
                <div v-for="pf in personaFilesMap[editingPersona.id]" :key="pf.id" class="flex items-center justify-between bg-gray-50 dark:bg-gray-750 rounded-lg px-3 py-2">
                  <div class="flex-1 min-w-0">
                    <p class="text-xs font-medium text-gray-700 dark:text-gray-300 truncate">{{ pf.file_name }}</p>
                    <div class="flex items-center gap-1.5 mt-0.5">
                        <span class="text-xs px-1.5 py-0.5 rounded-full" :class="moduleBadgeClass(pf.module_category)">{{ pf.module_category }}</span>
                        <span class="text-xs text-gray-400">优先级 {{ pf.priority }}</span>
                      </div>
                  </div>
                  <button class="text-xs text-red-400 hover:text-red-600 shrink-0 ml-2" @click="deletePersonaFile(editingPersona.id, pf.id)">删除</button>
                </div>
              </div>
              <p v-else class="text-xs text-gray-400 py-2">暂无技能文件</p>
            </div>
          </div>
          <div class="px-5 py-4 border-t border-gray-100 dark:border-gray-700 flex justify-between gap-2">
            <button v-if="editingPersona && !editingPersona.is_built_in"
              class="text-sm text-red-400 border border-red-200 dark:border-red-800 rounded-lg px-4 py-2 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
              @click="deletePersona(editingPersona.id)"
            >删除人格</button>
            <div class="flex gap-2 ml-auto">
              <button class="text-sm text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-750 rounded-lg px-4 py-2 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors" @click="editingPersona = null">取消</button>
              <button class="text-sm text-white bg-wechat-green hover:bg-wechat-green-dark rounded-lg px-4 py-2 font-medium transition-colors disabled:opacity-50"
                :disabled="!editPersonaName.trim() || submittingEditPersona"
                @click="submitEditPersona">{{ submittingEditPersona ? '提交中...' : '保存' }}</button>
            </div>
          </div>
        </div>
      </div>
    </Transition>

    <!-- New Conversation Modal -->
    <Transition name="fade">
      <div v-if="showNewConvModal" class="fixed inset-0 z-50 flex items-center justify-center bg-black/40" @click.self="showNewConvModal = false">
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-full max-w-sm mx-4 overflow-hidden" @click.stop>
          <div class="px-5 py-4 border-b border-gray-100 dark:border-gray-700">
            <h3 class="text-base font-semibold text-gray-800 dark:text-gray-100">选择人格</h3>
            <p class="text-xs text-gray-400 mt-0.5">选择一个角色开始对话，或使用默认人格</p>
          </div>
          <div class="max-h-72 overflow-y-auto">
            <div v-for="p in personas" :key="p.id"
              class="flex items-center gap-3 px-5 py-3 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-750 transition-colors border-b border-gray-100 dark:border-gray-700/50"
              @click="handleNewConversation(p)"
            >
              <div class="w-10 h-10 rounded-full shrink-0 overflow-hidden" :style="{ backgroundColor: personaColor(p.name) }">
                <img v-if="p.avatar" :src="p.avatar" class="w-full h-full object-cover" />
                <div v-else class="w-full h-full flex items-center justify-center text-white text-sm font-bold">{{ p.name.charAt(0).toUpperCase() }}</div>
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium text-gray-800 dark:text-gray-200">{{ p.nickname || p.name }}</p>
                <p class="text-xs text-gray-400 truncate">{{ p.description || '暂无描述' }}</p>
              </div>
            </div>
          </div>
          <div class="px-5 py-3 border-t border-gray-100 dark:border-gray-700 flex gap-2">
            <button class="flex-1 text-sm text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-750 rounded-lg py-2 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors" @click="handleNewConversation(null)">默认人格</button>
            <button class="flex-1 text-sm text-gray-500 border border-gray-200 dark:border-gray-600 rounded-lg py-2 hover:bg-gray-50 dark:hover:bg-gray-750 transition-colors" @click="showNewConvModal = false">取消</button>
          </div>
        </div>
      </div>
    </Transition>

    <EmojiPicker :visible="showEmojiPicker" @close="showEmojiPicker = false" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useChatStore } from '../store/chat'
import { useUserStore } from '../store/user'
import { useThemeStore } from '../store/theme'
import ChatBubble from '../components/ChatBubble.vue'
import TimeStamp from '../components/TimeStamp.vue'
import EmojiPicker from '../components/EmojiPicker.vue'
import VoiceButton from '../components/VoiceButton.vue'
import { personaAPI } from '../api'
import { formatConversationTime, truncate, sanitizeHtml } from '../utils'
import type { Conversation, Message, PersonaFromServer, PersonaFile } from '../types/api'

const tabs = [
  { key: 'chat', label: '对话', icon: '<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>' },
  { key: 'persona', label: '人格', icon: '<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>' },
  { key: 'my', label: '我的', icon: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/>' },
]

const chatStore = useChatStore()
const userStore = useUserStore()
const themeStore = useThemeStore()

const activeTab = ref('chat')
const isMobile = ref(window.innerWidth < 1024)

// Chat
const searchQuery = ref('')
const inputContent = ref('')
const showSidebar = ref(true)
const showEmojiPicker = ref(false)
const aiAvatarError = ref(false)
const showNewConvModal = ref(false)
const messagesContainer = ref<HTMLElement | null>(null)
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const loadMoreSentinel = ref<HTMLElement | null>(null)
const bottomSentinel = ref<HTMLElement | null>(null)
let loadMoreObserver: IntersectionObserver | null = null
let isAutoScrolling = false

// Settings
const showSuccess = ref(false)
const successMessage = ref('')
const uploadingAvatarFor = ref<string | null>(null)
const debugExpandedMsgId = ref<number | null>(null)
const debugMsgContent = ref('')
const debugSystemCache = ref('')

// Persona list (shared between Persona tab and My tab)
const personas = ref<PersonaFromServer[]>([])
const personaFilesMap = ref<Record<number, PersonaFile[]>>({})
const personaSearchQuery = ref('')

const filteredPersonas = computed(() => {
  if (!personaSearchQuery.value) return personas.value
  const q = personaSearchQuery.value.toLowerCase()
  return personas.value.filter(p => p.name.toLowerCase().includes(q))
})

// Add Persona Modal
const showAddPersonaModal = ref(false)
const addPersonaSystemName = ref('')
const addPersonaNickname = ref('')
const addPersonaFiles = ref<File[]>([])
const addPersonaFileErrors = ref<string[]>([])
const submittingAddPersona = ref(false)

// Edit Persona Modal
const editingPersona = ref<PersonaFromServer | null>(null)
const editPersonaName = ref('')
const editPersonaDesc = ref('')
const editPersonaAvatarUrl = ref('')
const submittingEditPersona = ref(false)
const uploadingPersonaAvatar = ref(false)

const filteredConversations = computed(() => {
  if (!searchQuery.value) return chatStore.conversations
  const q = searchQuery.value.toLowerCase()
  return chatStore.conversations.filter(c => (c.title || '').toLowerCase().includes(q))
})

// ===== Chat Functions =====
function shouldShowTimestamp(msg: Message, idx: number) {
  if (idx === 0) return true
  const prev = chatStore.messages[idx - 1]
  if (!prev || !prev.created_at || !msg.created_at) return true
  return new Date(msg.created_at).getTime() - new Date(prev.created_at).getTime() > 5 * 60 * 1000
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); sendMessage() }
}

function sendMessage() {
  const text = inputContent.value.trim()
  if (!text || chatStore.isStreaming) return
  chatStore.sendMessage(sanitizeHtml(text))
  inputContent.value = ''
  autoGrow()
  scrollToBottom()
}

function autoGrow() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  const lineHeight = parseInt(getComputedStyle(el).lineHeight) || 20
  el.style.height = Math.min(el.scrollHeight, lineHeight * 4 + 8) + 'px'
}

function selectConversation(conv: Conversation) {
  if (conv.id === chatStore.currentConversationId) return
  chatStore.selectConversation(conv.id)
  aiAvatarError.value = false
  if (isMobile.value) showSidebar.value = false
  resetTextarea()
}

async function handleNewConversation(persona: PersonaFromServer | null) {
  showNewConvModal.value = false
  try {
    const title = persona ? persona.name : '新对话'
    const conv = await chatStore.createConversation()
    if (persona) {
      await chatStore.updateConversationConfig(conv.id, { persona_id: persona.id, ai_avatar: persona.avatar || conv.ai_avatar, title })
    }
    chatStore.selectConversation(conv.id)
    aiAvatarError.value = false
    if (isMobile.value) showSidebar.value = false
    resetTextarea()
  } catch (e) { console.error('创建会话失败:', e) }
}

function resetTextarea() { inputContent.value = ''; autoGrow() }

function scrollToBottom() {
  isAutoScrolling = true
  nextTick(() => { bottomSentinel.value?.scrollIntoView({ behavior: 'smooth' }); setTimeout(() => { isAutoScrolling = false }, 300) })
}

function checkMobile() {
  const wasMobile = isMobile.value
  isMobile.value = window.innerWidth < 1024
  if (!isMobile.value && wasMobile) showSidebar.value = true
}

const toggleMsgDebug = async (msgId: number, msgIdx: number) => {
  if (debugExpandedMsgId.value === msgId) { debugExpandedMsgId.value = null; return }

  if (!debugSystemCache.value && chatStore.currentConversationId) {
    try {
      const res = await personaAPI.getDebugPrompt(chatStore.currentConversationId)
      debugSystemCache.value = res.system_prompt || ''
    } catch { debugSystemCache.value = '' }
  }

  const messages = chatStore.messages.slice(0, msgIdx + 1)
  const history = messages.map(m => {
    const role = m.role === 'assistant' ? 'Assistant' : 'User'
    return `${role}: ${m.content}`
  }).join('\n\n')

  const full = `=== System Prompt ===\n${debugSystemCache.value}\n\n=== Conversation Context ===\n${history}`
  debugMsgContent.value = full
  debugExpandedMsgId.value = msgId
}

// ===== Settings Functions =====
function showToast(msg: string) { successMessage.value = msg; showSuccess.value = true; setTimeout(() => { showSuccess.value = false }, 2000) }

async function fetchPersonas() {
  try { const res = await personaAPI.getPersonas(); personas.value = res.personas || [] } catch {}
}

function moduleBadgeClass(category: string): string {
  const map: Record<string, string> = {
    persona_base: 'bg-pink-100 text-pink-600 dark:bg-pink-900/30 dark:text-pink-400',
    persona_tone: 'bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400',
    forbidden_rules: 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400',
    emotion_companion: 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400',
    professional_skills: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400',
    style_switch: 'bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400',
    trigger_rules: 'bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400',
  }
  return map[category] || 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
}

const personaColors = ['#6366f1', '#8b5cf6', '#ec4899', '#f43f5e', '#f97316', '#eab308', '#22c55e', '#14b8a6', '#06b6d4', '#3b82f6']
function personaColor(name: string): string {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return personaColors[Math.abs(hash) % personaColors.length]
}

async function openPersonaSettings(p: PersonaFromServer) {
  openEditPersona(p)
  try { const res = await personaAPI.getPersona(p.id); personaFilesMap.value[p.id] = res.persona_files || [] } catch { personaFilesMap.value[p.id] = [] }
}

async function switchPersona(personaId: number | null) {
  if (!chatStore.currentConversationId) { showToast('请先选择一个会话'); return }
  try {
    await chatStore.setConversationPersona(personaId)
    showToast('人格已切换')
  } catch { showToast('切换失败') }
}

async function switchToPersona(personaId: number) {
  if (!chatStore.currentConversationId) { showToast('请先选择一个会话'); return }
  try {
    await chatStore.setConversationPersona(personaId)
    showToast('人格已切换')
    activeTab.value = 'chat'
  } catch { showToast('切换失败') }
}

async function deletePersona(id: number) {
  if (!window.confirm('确定要删除该人格及其所有文件吗？')) return
  try {
    await personaAPI.deletePersona(id)
    showToast('人格已删除')
    editingPersona.value = null
    await fetchPersonas()
  } catch { showToast('删除失败') }
}

// ===== Edit Persona =====
function openEditPersona(p: PersonaFromServer) {
  editingPersona.value = p
  editPersonaName.value = p.nickname || p.name
  editPersonaDesc.value = p.description || ''
  editPersonaAvatarUrl.value = p.avatar || ''
  submittingEditPersona.value = false
}

async function submitEditPersona() {
  if (!editingPersona.value || !editPersonaName.value.trim()) return
  submittingEditPersona.value = true
  try {
    await personaAPI.updatePersona(editingPersona.value.id, { nickname: editPersonaName.value.trim(), description: editPersonaDesc.value.trim() })
    showToast('人格已更新')
    editingPersona.value = null
    await fetchPersonas()
  } catch { showToast('更新失败') } finally { submittingEditPersona.value = false }
}

async function uploadPersonaAvatar(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !editingPersona.value) return
  if (file.size > 1024 * 1024) { showToast('图片不能超过 1MB'); input.value = ''; return }
  uploadingPersonaAvatar.value = true
  try {
    const res = await personaAPI.uploadPersonaAvatar(editingPersona.value.id, file)
    editPersonaAvatarUrl.value = res.avatar
    editingPersona.value.avatar = res.avatar
    showToast('头像已更新')
  } catch { showToast('上传失败') } finally { uploadingPersonaAvatar.value = false; input.value = '' }
}

async function uploadPersonaSkillFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0 || !editingPersona.value) return
  try {
    const res = await personaAPI.uploadSkillFile(editingPersona.value.id, Array.from(files))
    showToast(`成功上传 ${res.uploaded?.length || 0} 个文件`)
    const detail = await personaAPI.getPersona(editingPersona.value.id)
    personaFilesMap.value[editingPersona.value.id] = detail.persona_files || []
  } catch { showToast('上传失败') }
  input.value = ''
}

async function deletePersonaFile(personaId: number, fileId: number) {
  if (!window.confirm('确定要删除该文件吗？')) return
  try {
    await personaAPI.deleteSkillFile(personaId, fileId)
    const detail = await personaAPI.getPersona(personaId)
    personaFilesMap.value[personaId] = detail.persona_files || []
  } catch { showToast('删除失败') }
}

async function uploadUserAvatar(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  if (file.size > 1024 * 1024) { showToast('图片大小不能超过 1MB'); input.value = ''; return }
  uploadingAvatarFor.value = 'user'
  try {
    const res = await personaAPI.uploadAvatar(file)
    if (userStore.userInfo) (userStore.userInfo as any).avatar = res.file.url
    showToast('头像已更新')
  } catch { showToast('上传失败') } finally { uploadingAvatarFor.value = null }
  input.value = ''
}

// ===== Add Persona Modal =====
function openAddPersonaModal() {
  addPersonaSystemName.value = ''
  addPersonaNickname.value = ''
  addPersonaFiles.value = []
  addPersonaFileErrors.value = []
  submittingAddPersona.value = false
  showAddPersonaModal.value = true
}
function closeAddPersonaModal() { showAddPersonaModal.value = false }

function onAddPersonaFilesChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return
  const errors: string[] = []
  const valid: File[] = []
  for (const f of Array.from(files)) {
    if (!f.name.endsWith('.md')) { errors.push(`跳过非 .md 文件: ${f.name}`); continue }
    if (f.size > 10 * 1024) { errors.push(`${f.name} 超过 10KB (${(f.size / 1024).toFixed(1)}KB)`); continue }
    valid.push(f)
  }
  addPersonaFiles.value = [...addPersonaFiles.value, ...valid]
  addPersonaFileErrors.value = errors
  input.value = ''
}

function removeAddPersonaFile(index: number) { addPersonaFiles.value.splice(index, 1) }

async function submitAddPersona() {
  const systemName = addPersonaSystemName.value.trim()
  const nickname = addPersonaNickname.value.trim()
  if (!systemName || !nickname || addPersonaFiles.value.length === 0) return
  submittingAddPersona.value = true
  try {
    const res = await personaAPI.createPersona({ name: systemName, nickname, description: `自定义人格: ${nickname}` })
    await personaAPI.uploadSkillFile(res.persona.id, addPersonaFiles.value)
    showToast(`人格 "${nickname}" 创建成功`)
    closeAddPersonaModal()
    await fetchPersonas()
  } catch { showToast('添加失败') } finally { submittingAddPersona.value = false }
}

// ===== Logout =====
function handleLogout() {
  userStore.logout()
  window.location.href = '/login'
}

// ===== Watchers =====
watch(() => chatStore.messages.length, () => { if (!chatStore.isLoading) scrollToBottom() })
watch(() => chatStore.streamingContent, () => { if (chatStore.isStreaming) scrollToBottom() })
watch(() => chatStore.isStreaming, (val) => { if (!val) nextTick(() => scrollToBottom()) })
watch(() => chatStore.messages.length, () => { observeLoadMoreSentinel() })
watch(() => chatStore.currentConversationId, () => { aiAvatarError.value = false })

// ===== Lifecycle =====
onMounted(async () => {
  window.addEventListener('resize', checkMobile)
  try { await chatStore.fetchConversations() } catch {}
  if (chatStore.conversations.length > 0 && !chatStore.currentConversationId) {
    await chatStore.selectConversation(chatStore.conversations[0].id)
  }
  aiAvatarError.value = false
  chatStore.connectWebSocket()
  nextTick(() => { scrollToBottom(); autoGrow() })
  loadMoreObserver = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && !chatStore.isLoading && chatStore.messages.length > 0) chatStore.loadMoreMessages()
  }, { threshold: 0.1 })
  observeLoadMoreSentinel()
  fetchPersonas()
})

function observeLoadMoreSentinel() {
  nextTick(() => { if (loadMoreSentinel.value && loadMoreObserver) { loadMoreObserver.disconnect(); loadMoreObserver.observe(loadMoreSentinel.value) } })
}

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
  if (loadMoreObserver) { loadMoreObserver.disconnect(); loadMoreObserver = null }
  chatStore.disconnectWebSocket()
})
</script>

<style scoped>
textarea::-webkit-scrollbar { width: 4px; }
textarea::-webkit-scrollbar-thumb { background: #c1c1c1; border-radius: 2px; }
.dark textarea::-webkit-scrollbar-thumb { background: #4a4a4a; }
.fade-enter-active, .fade-leave-active { transition: opacity 0.2s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
