# ?? HitUp — Real-Time Distributed Cloud Messenger

HitUp is a high-performance, real-time distributed messaging application built with **Go**, **WebSockets**, **PostgreSQL**, **Redis Pub/Sub**, and **WebRTC**.

## ?? Features

- **Authentication & Security:** JWT-based sessions, BCrypt password hashing, 6-digit Email OTP verification, and Google OAuth 2.0.
- **Real-Time Engine:** Low-latency WebSockets with Goroutines/Channels and horizontal scaling via Redis Pub/Sub.
- **Rich Messaging:** 1:1 Direct chats, Group chats with Admin roles & Invite codes, Ephemeral/Disappearing messages (?), Starred messages (?), and In-chat search (??).
- **Interactive Mechanics:** Emoji reactions (?? ?? ?? ??), Reply/Quote preview, Forwarding, In-place Edit, and Delete for Everyone.
- **Media & Voice:** Photo/Video/Document sharing, Voice notes (MediaRecorder API), GPS location cards, and OpenGraph link previews.
- **Voice & Video Calls:** WebRTC P2P 1:1 Audio/Video calls with screen sharing.
- **24h Stories / Status:** Ephemeral stories with viewer tracking.
- **Themes & UI:** Modern WhatsApp-inspired Dark & Light modes.

## ??? Tech Stack

- **Backend:** Go (Golang)
- **Database:** PostgreSQL (Neon.tech / Local)
- **Cache & Pub/Sub:** Redis Cloud / Local
- **Signaling & Media:** WebSockets & WebRTC
- **Reverse Proxy:** Nginx Load Balancer
- **Deployment:** Docker & Docker Compose / Render.com
