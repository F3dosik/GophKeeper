<script lang="ts">
  import {CheckServer, ChooseCACert} from '../wailsjs/go/backend/App.js'

  let address = "localhost:50051"
  let caPEM = ""
  let result = ""
  let error = ""

  async function check(): Promise<void> {
    result = ""
    error = ""
    try {
      result = await CheckServer(address, caPEM)
    } catch (e) {
      error = String(e)
    }
  }
</script>

<main>
  <h1>GophKeeper</h1>
  <label>Адрес сервера <input bind:value={address} autocomplete="off"/></label>
  <button on:click={async () => caPEM = await ChooseCACert()}>Выбрать ca.crt {caPEM ? "✓" : ""}</button>
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
