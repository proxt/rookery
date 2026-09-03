// Mirrors the iota ordering of engine.State / engine.EventType in
// client/internal/engine/engine.go — JSON encodes them as plain numbers.
export const State = {
  DISCONNECTED: 0,
  CONNECTING: 1,
  CONNECTED: 2,
  ERROR: 3,
}

export const EventType = {
  STATE_CHANGED: 0,
  STATS_TICK: 1,
  KILL_SWITCH_WARNING: 2,
}

export const STATE_LABEL = {
  [State.DISCONNECTED]: 'Отключено',
  [State.CONNECTING]: 'Подключение',
  [State.CONNECTED]: 'Подключено',
  [State.ERROR]: 'Ошибка',
}
