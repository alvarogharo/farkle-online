import { ref, onUnmounted } from 'vue'
import { WS_URL } from '@/config.js'

/**
 * Composable para gestión de conexión WebSocket con el backend Farkle.
 * Incluye reconexión automática básica y heartbeat para detectar sockets "zombi"
 * (típicos de dispositivos móviles al ir a segundo plano).
 *
 * @param {Object} options
 * @param {string} options.url - URL del WebSocket (default: desde config/ env VITE_WS_URL)
 * @param {boolean} options.autoReconnect - Si reintentar conexión al cerrarse (default: true)
 * @param {number} options.maxRetries - Intentos máximos de reconexión (default: 5)
 */
export function useWebSocket(options = {}) {
  const {
    url = WS_URL,
    autoReconnect = true,
    maxRetries = 5,
  } = options

  const connected = ref(false)
  const lastError = ref(null)
  const retryCount = ref(0)

  let ws = null
  let messageHandlers = []
  let reconnectTimeout = null
  let heartbeatIntervalId = null
  let lastPongAt = 0

  const HEARTBEAT_INTERVAL_MS = 15000
  const HEARTBEAT_STALE_MS = 40000

  function startHeartbeat() {
    if (heartbeatIntervalId) {
      clearInterval(heartbeatIntervalId)
      heartbeatIntervalId = null
    }
    heartbeatIntervalId = setInterval(() => {
      if (!ws || ws.readyState !== WebSocket.OPEN) return

      const now = Date.now()
      if (lastPongAt && now - lastPongAt > HEARTBEAT_STALE_MS) {
        // Conexión estancada: forzar cierre para que la lógica de reconexión actúe
        ws.close()
        return
      }

      // Heartbeat ligero; el backend responde con msgPong
      try {
        ws.send(JSON.stringify({ type: 'ping' }))
      } catch {
        // Si falla el envío, dejamos que onclose gestione la reconexión
      }
    }, HEARTBEAT_INTERVAL_MS)
  }

  function connect() {
    if (ws?.readyState === WebSocket.OPEN) return

    lastError.value = null
    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      retryCount.value = 0
      lastPongAt = Date.now()
      startHeartbeat()
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data?.type === 'pong') {
          lastPongAt = Date.now()
        }
        messageHandlers.forEach((fn) => fn(data))
      } catch (e) {
        console.error('[WebSocket] Error parsing message:', e)
      }
    }

    ws.onerror = () => {
      lastError.value = 'Connection error'
    }

    ws.onclose = (event) => {
      connected.value = false
      ws = null

      if (heartbeatIntervalId) {
        clearInterval(heartbeatIntervalId)
        heartbeatIntervalId = null
      }

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
    if (heartbeatIntervalId) {
      clearInterval(heartbeatIntervalId)
      heartbeatIntervalId = null
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
