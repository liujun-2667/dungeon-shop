<script>
  import { Router, Route } from 'svelte-routing';
  import Lobby from './pages/Lobby.svelte';
  import GameRoom from './pages/GameRoom.svelte';
  import Settlement from './pages/Settlement.svelte';
</script>

<Router>
  <Route path="/"><Lobby /></Route>
  <Route path="/room/:roomId" let:params>
    <GameRoom {params} />
  </Route>
  <Route path="/room/:roomId/settlement" let:params>
    <Settlement {params} />
  </Route>
</Router>
