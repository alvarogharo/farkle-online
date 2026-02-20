/**
 * Configuración centralizada del frontend Farkle.
 * Las variables de entorno (VITE_*) se leen en build time.
 * Ver .env.example para las variables disponibles.
 */

// ─── WebSocket ─────────────────────────────────────────────────────────────
const WS_DEFAULT_URL = 'ws://localhost:8080/ws';
export const WS_URL = import.meta.env.VITE_WS_URL || WS_DEFAULT_URL;

// ─── Animación y timing ────────────────────────────────────────────────────
export const ROLL_DISPLAY_MS = 1200;
export const ROLL_ANIMATION_INTERVAL_MS = 80;
export const TOAST_DELAY_MS = 200;
export const TOAST_DURATION_MS = 2000;

// ─── Dados ─────────────────────────────────────────────────────────────────
export const DICE_SIDES = 6;
export const DEFAULT_DICE_COUNT = 6;

// ─── Puntuación (lobby y validación) ───────────────────────────────────────
export const DEFAULT_VICTORY_SCORE = 2000;
export const MIN_VICTORY_SCORE = 100;
export const MAX_VICTORY_SCORE = 100000;

// ─── Tipos de mensajes WebSocket (recibidos del backend) ───────────────────
export const MSG = {
  GAME_STATE: 'game_state',
  ROLL_RESULT: 'roll_result',
  ERROR: 'error',
  FARKLE: 'farkle',
  HOT_DICE: 'hot_dice',
  TURN_CHANGED: 'turn_changed',
  FINAL_ROUND: 'final_round',
  GAME_OVER: 'game_over',
  PLAYER_DISCONNECTED: 'player_disconnected',
};

// ─── Tipos de mensajes WebSocket (enviados al backend) ──────────────────────
export const MSG_SEND = {
  ROLL: 'roll',
  TOGGLE_SELECT: 'toggle_select',
  APARTAR: 'apartar',
  BANK: 'bank',
  CREATE: 'create',
  JOIN: 'join',
};

// ─── Tipos de mensajes WebSocket (lobby) ────────────────────────────────────
export const MSG_LOBBY = {
  ERROR: 'error',
  GAME_CREATED: 'game_created',
  PLAYER_JOINED: 'player_joined',
  GAME_JOINED: 'game_joined',
};
