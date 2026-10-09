<script lang="ts">
  import {CheckServer} from '../wailsjs/go/main/App.js'

  let address = "localhost:50051"
  let caCertPath = ""
  let result = ""
  let error = ""

  async function check(): Promise<void> {
    result = ""
    error = ""
    try {
      result = await CheckServer(address, caCertPath)
    } catch (e) {
      error = String(e)
    }
  }
</script>

<main>
  <h1>GophKeeper</h1>
  <label>Адрес сервера <input bind:value={address} autocomplete="off"/></label>
  <label>Путь к ca.crt <input bind:value={caCertPath} autocomplete="off"/></label>
  <button on:click={check}>Проверить подключение</button>
  {#if result}<p class="ok">{result}</p>{/if}
  {#if error}<p class="err">{error}</p>{/if}
</main>

<style>
  main { display: flex; flex-direction: column; gap: 0.75rem; max-width: 32rem; margin: 3rem auto; text-align: left; }
  label { display: flex; flex-direction: column; gap: 0.25rem; }
  input { padding: 0.4rem; }
  .ok { color: #3c3; }
  .err { color: #e55; }
</style>
