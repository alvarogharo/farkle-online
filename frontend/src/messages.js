/**
 * Mensajes de texto de la UI.
 * Para strings con interpolación se usan funciones.
 */

// ─── Juego (App.vue) ────────────────────────────────────────────────────────
export const game = {
  farkleSelf: 'Farkle: pierdes los puntos del turno',
  farkleOther: (name) => `¡${name} pierde sus puntos por Farkle!`,
  hotDiceSelf: '¡Mano limpia! Puedes volver a tirar los 6 dados',
  hotDiceOther: (name) => `¡${name} ha conseguido mano limpia!`,
  turnYourTurn: '¡Tu turno!',
  turnChanged: 'Cambio de turno',
  finalRoundTriggered: 'Has alcanzado la meta. El otro jugador tiene un último turno.',
  finalRoundLastTurn: '¡Ronda final! Este es tu último turno.',
  finalRoundBannerTriggered: 'Has alcanzado la meta. El otro jugador tiene un último turno.',
  finalRoundBannerOther: (name) => `Último turno de ${name}.`,
  gameOverYouWin: '¡Ganas la partida!',
  gameOverOtherWins: (name) => `${name} gana la partida`,
  gameOverFinished: 'Partida terminada',
  disconnectYouWin: 'El otro jugador se ha desconectado. Ganas la partida.',
  disconnectOverlay: 'El otro jugador se ha desconectado. Ganas la partida por abandono.',
  youWin: '¡Has ganado!',
  youLose: 'Has perdido',
  diceReady: (n) => `Tienes ${n} dados listos. Pulsa "Tirar dados".`,
  waitingForTurn: (name) => `Es el turno de ${name}. Esperando su tirada…`,
  gameOverTitle: 'Partida terminada',
  backToLobby: 'Volver al lobby',
  yourTurn: 'Tu turno',
  turnOf: (name) => `Turno de ${name}`,
  rolling: 'Tirando...',
  rollDice: 'Tirar dados',
  apartar: 'Apartar',
  bank: 'Plantarse',
  savedSection: 'Apartados del turno',
  lotLabel: (id) => `Lote ${id}`,
  errorDefault: 'Error',
  playerDefault: 'Jugador',
  playerFallback: 'el jugador',
};

// ─── Lobby (LobbyModal.vue) ────────────────────────────────────────────────
export const lobby = {
  serverError: 'Error del servidor',
  codeRequired: 'Introduce el código de la partida',
  defaultPlayer1: 'Jugador 1',
  defaultPlayer2: 'Jugador 2',
  shareHint: 'Comparte este código o enlace para que alguien se una a la partida:',
  copyCode: 'Copiar código',
  copyCodeDone: '¡Copiado!',
  copyLink: 'Copiar enlace',
  copyLinkDone: '¡Enlace copiado!',
  waitingHint: 'Esperando que alguien se una…',
  joinLinkIntro: 'Has llegado mediante un enlace. Introduce tu nombre para unirte a la partida.',
  yourName: 'Tu nombre',
};
