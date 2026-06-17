<script>
  import { Router, Route } from 'svelte-routing';
  import Lobby from './pages/Lobby.svelte';
  import GameRoom from './pages/GameRoom.svelte';
  import Settlement from './pages/Settlement.svelte';
</script>

<Router>
  <Route path="/" component={Lobby} />
  <Route path="/room/:roomId" component={GameRoom} />
  <Route path="/room/:roomId/settlement" component={Settlement} />
</Router>
