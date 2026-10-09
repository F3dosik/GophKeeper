<script lang="ts">
  // Поле пароля с кнопкой «Показать». Значение передаётся через bind:value.
  let {
    value = $bindable(''),
    placeholder = '',
    autofocus = false,
    name = '',
  }: { value?: string; placeholder?: string; autofocus?: boolean; name?: string } = $props()

  let shown = $state(false)
</script>

<div class="pw">
  <!-- svelte-ignore a11y_autofocus -->
  <input
    type={shown ? 'text' : 'password'}
    {value}
    {placeholder}
    {name}
    {autofocus}
    autocomplete="off"
    spellcheck="false"
    oninput={(e) => (value = e.currentTarget.value)}
  />
  <button type="button" class="link" tabindex="-1" onclick={() => (shown = !shown)}>{shown ? 'Скрыть' : 'Показать'}</button>
</div>

<style>
  .pw { position: relative; }
  .pw input { padding-right: 84px; }
  .pw button { position: absolute; right: 6px; top: 50%; transform: translateY(-50%); font-size: 12px; }
</style>
