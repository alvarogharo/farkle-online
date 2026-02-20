import { ref, onUnmounted } from 'vue'

const DEFAULT_URL = 'ws://172.20.6.189:8080/ws'

/**
 * Composable para gestión de conexión WebSocket con el backend Farkle.
 * @param {Object} options
 * @param {string} options.url - URL del WebSocket (default: ws://localhost:8080/ws)
 * @param {boolean} options.autoReconnect - Si reintentar conexión al cerrarse (default: true)
 * @param {number} options.maxRetries - Intentos máximos de reconexión (default: 5)
 */
export function useWebSocket(options = {}) {
  const {
    url = DEFAULT_URL,
    autoReconnect = true,
    maxRetries = 5,
  } = options

  const connected = ref(false)
  const lastError = ref(null)
  const retryCount = ref(0)

  let ws = null
  let messageHandlers = []
  let reconnectTimeout = null

  function connect() {
    if (ws?.readyState === WebSocket.OPEN) return

    lastError.value = null
    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      retryCount.value = 0
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        messageHandlers.forEach((fn) => fn(data))
      } catch (e) {
        console.error('[WebSocket] Error parsing message:', e)
      }
    }

    ws.onerror = () => {
      lastError.value = 'Error de conexión'
    }

    ws.onclose = (event) => {
      connected.value = false
      ws = null

      if (autoReconnect && retryCount.value < maxRetries && !event.wasClean) {
        const delay = Math.min(1000 * 2 ** retryCount.value, 30000)
        retryCount.value++
        reconnectTimeout = setTimeout(connect, delay)
      }
    }
  }

  function disconnect() {
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout)
      reconnectTimeout = null
    }
    retryCount.value = maxRetries // Evita reconexión al desconectar manualmente
    if (ws) {
      ws.close()
      ws = null
    }
    connected.value = false
  }

  function send(obj) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.warn('[WebSocket] No conectado, no se puede enviar:', obj)
      return false
    }
    ws.send(JSON.stringify(obj))
    return true
  }

  function onMessage(handler) {
    if (typeof handler === 'function') {
      messageHandlers.push(handler)
      return () => {
        messageHandlers = messageHandlers.filter((h) => h !== handler)
      }
    }
    return () => {}
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    connected,
    lastError,
    connect,
    disconnect,
    send,
    onMessage,
  }
}
