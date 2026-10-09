<script lang="ts">
  import { api, errorText } from '../lib/api'

  // Кнопка «Предложить парольную фразу» для мастер-пароля. Фраза показывается открыто,
  // чтобы её можно было запомнить, и подставляется в поля пароля через onpick.
  let { onpick }: { onpick: (phrase: string) => void } = $props()

  let phrase = $state('')
  let bits = $state(0)
  let error = $state('')

  async function suggest() {
    error = ''
    try {
      const g = await api.generatePassphrase(6)
      phrase = g.password
      bits = g.entropyBits
      onpick(phrase)
    } catch (e) {
      error = errorText(e)
    }
  }
</script>

<div class="suggest">
  <button type="button" class="link" onclick={suggest}>{phrase ? 'Другая фраза' : 'Предложить парольную фразу'}</button>
  {#if phrase}
    <code>{phrase}</code>
    <p class="muted">
      {bits} бит. Запомните или запишите фразу и храните её отдельно от устройства.
    </p>
  {/if}
  {#if error}<p class="error">{error}</p>{/if}
</div>

<style>
  .suggest { display: flex; flex-direction: column; gap: 6px; }
  .suggest button { align-self: flex-start; padding-left: 0; }
  code { padding: 10px; background: var(--panel-2); border-radius: 6px; font-size: 15px; user-select: text; word-break: break-word; }
  p { margin: 0; font-size: 12px; }
</style>
