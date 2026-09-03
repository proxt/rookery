<script>
  import { fade } from 'svelte/transition'
  import { flip } from 'svelte/animate'
  import { api } from '../api.js'
  import { formatDateTime } from '../format.js'

  let entries = $state([])
  let loading = $state(true)

  const ACTION_LABELS = {
    'auth.login': 'Вход в панель',
    'settings.update': 'Изменение настроек',
    'account.password.change': 'Смена пароля',
    'admin.create': 'Создан администратор',
    'admin.delete': 'Удалён администратор',
    'user.create': 'Создан пользователь',
    'user.update': 'Изменён пользователь',
    'user.delete': 'Удалён пользователь',
    'user.nodes.update': 'Изменён доступ к нодам',
    'node.create': 'Создана нода',
    'node.update': 'Изменена нода',
    'node.delete': 'Удалена нода',
    'release.upload': 'Загружен релиз',
    'release.delete': 'Удалён релиз',
  }

  function actionLabel(action) {
    return ACTION_LABELS[action] ?? action
  }

  async function load() {
    loading = true
    entries = (await api.listAuditLog()) ?? []
    loading = false
  }
  load()
</script>

<div class="mx-auto max-w-4xl">
  <div class="mb-6 flex items-center justify-between">
    <h1 class="text-lg font-semibold">Журнал действий</h1>
    <button class="btn-secondary" onclick={load}>Обновить</button>
  </div>
  <p class="mb-4 text-xs text-muted">Последние {entries.length} записей. Хранится 90 дней, старые записи удаляются автоматически.</p>

  {#if loading}
    <div class="py-16 text-center text-sm text-muted">Загрузка…</div>
  {:else if entries.length === 0}
    <div class="card fade-in-up py-16 text-center text-sm text-muted">Пока нет ни одной записи</div>
  {:else}
    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border text-left text-xs uppercase tracking-wide text-muted">
            <th class="px-4 py-3 font-medium">Время</th>
            <th class="px-4 py-3 font-medium">Админ</th>
            <th class="px-4 py-3 font-medium">Действие</th>
            <th class="px-4 py-3 font-medium">Цель</th>
            <th class="px-4 py-3 font-medium">Детали</th>
          </tr>
        </thead>
        <tbody>
          {#each entries as e (e.id)}
            <tr
              class="border-b border-border/60 last:border-0"
              animate:flip={{ duration: 200 }}
              transition:fade={{ duration: 150 }}
            >
              <td class="whitespace-nowrap px-4 py-3 text-muted">{formatDateTime(e.created_at)}</td>
              <td class="px-4 py-3 font-medium">{e.admin_name}</td>
              <td class="px-4 py-3">{actionLabel(e.action)}</td>
              <td class="px-4 py-3 text-muted">{e.target_type ? `${e.target_type} · ${e.target_id.slice(0, 8)}` : '—'}</td>
              <td class="max-w-xs truncate px-4 py-3 text-muted" title={e.detail}>{e.detail || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
