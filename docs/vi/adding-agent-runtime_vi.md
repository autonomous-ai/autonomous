# Thêm một Agentic Runtime Backend

Cách thêm một "bộ não" agent mới (như OpenClaw, Hermes, PicoClaw) vào OS, và
**mọi thứ nó phải nối** để đạt parity. Viết từ bài học port Hermes — nơi quá
nhiều mảnh bị bỏ thành no-op âm thầm và cắn lại sau này (config không sống qua
factory-reset, skills mất không quay lại, đổi tên identity chẳng làm gì, skills
không tự cập nhật).

> **Một quy tắc duy nhất phải nhớ:** OS (`os-server`) là nền tảng; backend là bộ
> não thay được. Bất cứ thứ gì OpenClaw làm mà người dùng thấy được, backend mới
> phải hoặc làm theo, hoặc **quyết định bỏ một cách có ý thức — và ghi lý do
> trong comment.** No-op là một quyết định, không bao giờ là mặc định.

Nguồn chân lý cho hợp đồng: `os/services/domain/agent.go` (interface
`AgentGateway`). Doc này giải thích phần *nào* quan trọng và *cách* nối switch,
install, migration, skills, hooks, reset.

---

## 0. Mô hình tư duy

- `config.agent_runtime` (`/root/config/config.json`) chọn backend đang chạy.
- `internal/agent/factory.go` `ProvideGateway` resolve lúc boot qua Wire DI:
  `config.agent_runtime` > DEVICE.md `gateway.default` > openclaw.
- Switch lúc runtime đi qua một core — `device.Service.UpdateAgentRuntime` — kích
  bởi 3 trigger (MQTT `agent_runtime.set`, HTTP `/api/device/agent-runtime`, web
  Runtime section). Xem `docs/vi/hermes_vi.md` §10–§11.

---

## 1. Hợp đồng — implement `domain.AgentGateway`

Backend nằm ở `internal/<name>/`, `*Service` của nó phải thoả **toàn bộ**
interface `AgentGateway`. Các method chia nhóm:

| Nhóm | Ví dụ | Lập trường backend mới |
|------|-------|------------------------|
| **Turn lõi** | `SendChatMessage`, `SendSystemChatMessage`, `*WithImage`, `NextChatRunID`, `*WithRun`, `StartWS` | **BẮT BUỘC** — đây là agent. |
| **Sẵn sàng / busy** | `IsReady`, `ConnectedAt`, `AgentUptime`, `IsBusy`, `SetBusy`, `QueuePendingEvent` | **BẮT BUỘC** — os-server gate sensing theo đây. |
| **Định danh** | `Name`, `Version`, `GetSessionKey`/`SetSessionKey` | **BẮT BUỘC** — hiện ở web Status. |
| **HAL passthrough** | `SendToHALTTS*`, `StopTTS`, `SetVolume`, `StartHALVoice` | Thường giống nhau giữa các backend — share hoặc copy. |
| **Run markers** | `MarkGuardRun`/`Consume*`, `MarkBroadcastRun`, `MarkPoseBucketRun`, `MarkWebChatRun`, `MarkSilentRun`, `*PendingChatTrace*` | **BẮT BUỘC track** — os-server gắn nhãn turn theo runID; OS phụ thuộc chúng. |
| **Kênh** | `AddChannel`, `RefreshChannelConfig`, `PairWhatsapp`, `HasWhatsappSession`, `GetTelegram*`, `Broadcast`, `SendToUser*` | Chỉ no-op khi backend thật sự không làm được (vd WhatsApp cần plugin Baileys). Telegram thường vẫn chạy qua config Lumi. |
| **Lifecycle / onboarding** | `SetupAgent`, `EnsureOnboarding`, `ResetAgent`, `RestartAgent`, `RefreshModelsConfig` | Quyết theo backend; ghi rõ lý do no-op. |
| **Cận-migration** | `UpdateIdentityName`, `StartSkillWatcher`, `WatchIdentity`, `StartModelSync`, `UpdatePrimaryModel`, `StartPrimaryModelWatch`, `CompactSession`, `NewSession`, `FetchChatHistory`, `WriteMCPEntry`/`RemoveMCPEntry`, `GetConfigJSON` | **Vùng nguy hiểm** — dễ no-op, đắt để phát hiện thiếu. Xem §4–§6. |

**Bài học (Hermes):** ~15 method bị stub no-op trong `internal/hermes/stubs.go`.
Một số đúng là N/A (`WriteMCPEntry` — không có `openclaw.json`; `PairWhatsapp` —
không có plugin). Nhưng `StartSkillWatcher`, `UpdateIdentityName`, và đường
config-sync là **gap chức năng**, không phải N/A — ship dạng no-op và chỉ phát
hiện khi skills bị cũ / đổi tên vô tác dụng / config gãy sau reset. Soát mọi
stub: ghi `// no-op because <lý do>` hoặc `// TODO(<backend>-<feature>)`, đừng để
thân hàm rỗng trơ.

---

## 2. Đăng ký + nối switch

1. `domain/device.go`: thêm const `AgentRuntime<Name>` + entry trong `AgentRuntimes`.
2. `internal/agent/factory.go`: thêm `case` trong `ProvideGateway`.
3. **Installer nhúng**: `internal/<name>/install.sh` + `install.go`
   (`//go:embed install.sh` → `runtimereg.Register(name, InstallScript)`).
4. `switch_runtime.sh` **generic** — không biết tên backend. **Đừng** sửa nó,
   imager, hay switch core của os-server để thêm backend.

Hợp đồng installer (`switch_runtime.sh` kỳ vọng):
- tạo systemd unit; khai tên nó ở `/usr/local/lib/os-runtimes/<name>/service` nếu
  khác `<name>.service`.
- tuỳ chọn drop `verify` hook ở `/usr/local/lib/os-runtimes/<name>/verify`
  (exit 0 = "đã cài & dùng được"). Giữ **rẻ** — xem §3 vì sao không được check quá tay.

---

## 3. Quy tắc vàng: install-một-lần vs mỗi-switch (*activation gap*)

`install.sh` chạy **một lần** — `switch_runtime.sh` chỉ chạy nó lúc cài đầu hoặc
khi `verify` fail. Mọi switch sau đều **bị skip**.

> **Do đó: bất cứ thứ gì phải sống qua factory-reset, hoặc phải refresh khi OTA
> `os-server` thường, KHÔNG được chỉ ghi bởi `install.sh`. Phải được os-server
> materialize MỖI switch.**

Đây là **activation gap** và Hermes dính 2 lần:
- Một fix nhét trong `install.sh` (hoặc bất kỳ file `install.sh` ghi ra — `verify`
  hook, presync hook) **không bao giờ tới được máy đã cài** qua OTA: bản cũ trên
  disk vẫn pass `verify`, nên `install.sh` không re-run, nên bản mới không bao giờ
  đáp.

Mẫu fix (dùng cho mọi thứ có state):
- Đặt logic vào **presync hook** (`runtime-<name>-presync`).
- Nhúng: `//go:embed presync.sh` → `runtimereg.RegisterPresync(name, PresyncScript)`.
- os-server materialize nó mỗi switch (`internal/device/runtime_installers.go`
  `materializePresync`, gọi trong switch flow cạnh `materializeInstaller`).
- `switch_runtime.sh` chạy `runtime-<name>-presync` ngay trước khi backend start.

Presync của Hermes (`internal/hermes/presync.sh`) giờ làm chủ **cả** model wiring
trong `config.yaml` (idempotent — coerce `model: ''` bị reset về map, khẳng định
structure `provider`/`custom_providers`, sync `llm_*`/secrets) **lẫn** restore
skills (chạy lại `claw migrate` khi `skills/openclaw-imports` rỗng). Giữ `verify`
chỉ-CLI (`command -v <bin>`) — structure-check trong `verify` sẽ ép full reinstall
nặng trong khi presync tự lành đủ rồi.

---

## 4. Migrate persona + memory (Go, chạy mỗi switch)

Thêm `internal/agent/migrate_persona/openclaw_to_<name>.go` và chiều ngược. Nó
chạy lúc os-server boot sau một switch thật (`Reconcile`, khi `agent_state.json`
prev ≠ current). Mang gì:

- **SOUL.md** → file identity của backend. Nếu backend **không có slot IDENTITY.md
  riêng** (Hermes không có), inline các field IDENTITY đã điền của owner thành
  block `## Your identity card` trong SOUL (xem `buildIdentityBlock`).
- **MEMORY.md + `memory/*.md` daily + KNOWLEDGE.md** → merge vào file long-term
  memory của backend. **Trước hết kiểm tra backend LOAD file nào THEO TÊN** —
  Hermes chỉ load `MEMORY.md` + `USER.md` (không glob `memories/*.md`), nên một
  `KNOWLEDGE.md` riêng sẽ bị bỏ qua; ta fold nó vào `MEMORY.md`.
- **USER.md** → file user-profile của backend.
- Đặt **`Overwrite = true`** cho copy soul khi switch: switch nghĩa là "lấy persona
  vừa dùng sang". `copyPersona` backup trước (`.bak-<nano>`).
- Chiều ngược phải **strip artifact riêng của backend** mà nó đã thêm (vd identity
  card — OpenClaw giữ tên trong IDENTITY.md riêng).

**ĐỪNG** mang file riêng-runtime: `AGENTS.md`, `TOOLS.md`, `HEARTBEAT.md`,
`hooks/` — chúng thuộc runtime nguồn. **Bộ nhớ sâu** của backend (DB
episodic/semantic, dream-diary, grounded-short-term) **không portable** — bản
distilled `MEMORY.md`/`USER.md` mới là dạng mang đi được.

---

## 5. Skills

- Skills tới backend bằng cách **copy** (kiểm tra: copy hay convert! `claw migrate`
  của Hermes là `shutil.copytree`, không transform) vào thư mục skill của backend.
- **Restore-sau-reset** thuộc **presync**, guard theo thư mục rỗng (để switch
  thường là no-op — không churn). Xem §3.
- **Skill watcher** (auto-update từ CDN, gate theo capability): plumbing generic
  fetch/extract/hash share ở `internal/skills/skillzip.go`
  (`FetchSkillVersions`/`DownloadToTempFile`/`FolderHash`/`ExtractSkillZip`). Thêm
  `internal/<name>/skill_watcher.go` mỏng song song với
  `internal/openclaw/skill_watcher.go` — chỉ khác **thư mục đích** và **đường
  notify**. Gate bằng `skills.Supported(device.Capabilities(...))`. Notify agent
  bằng `SendSystemChatMessage`.

---

## 6. Hooks — VẪN MỞ cho Hermes (ví dụ mẫu của một gap)

Hooks OpenClaw (`hooks/<name>/{HOOK.md, handler.ts}`) là handler TypeScript fire
trên event `message:preprocessed` của OpenClaw — `emotion-acknowledge` (mặt
"thinking" ngay khi nhận tin) và `turn-gate` (set busy cho turn từ kênh). Chúng
**riêng-runtime**, không portable.

Backend mới **không** thừa kế chúng. Lựa chọn:
- **Hook native của backend** — nếu backend có hệ hook (Hermes có Python plugin
  hook trên `pre_gateway_dispatch` / gateway `agent:start`, discover từ
  `~/.hermes/plugins`; **không có** loader drop-in `~/.hermes/hooks/HOOK.yaml`
  trong bản đang ship — verify loader thật của backend trước khi giả định).
- **Xử ở OS** — với thiết bị voice, hành vi giá trị nhất (mặt thinking lúc
  turn-start) nên làm trong `os-server` (Go), nơi phủ các turn do os-server tự
  khởi tạo (voice/sensing) bất kể backend. Turn từ kênh (Telegram) vẫn cần hook
  native của backend.

> Trạng thái: hooks Hermes **chưa implement**. Đây là gap parity còn lại sau khi
> skills/config/identity/persona đã đóng.

---

## 7. Factory reset

- Thêm wipe của backend ở `server/system/reset_<name>.go` (`wipe<Name>State`) và
  `case` trong `factoryreset.go`.
- **Wipe `/root/config/agent_state.json` khoá-bước với `config.json`** — chúng là
  một cặp (runtime hiện tại + lịch sử switch). Để lại `agent_state.json` trong khi
  `config.json` reset làm `prev` cũ lệch với `current` bị reset → kích **migration
  persona giả** lan state đã-wipe/stub.
- Giữ thứ phải sống (`bootstrap.json` = state OTA).
- Một wipe path xoá nội dung đã migrate (skills, config) phải có **restore path
  chạy SAU reset** (presync, §3/§5) — không thì nội dung mất luôn khi `install.sh`
  ngừng re-run.

---

## 8. Gate theo capability

Dùng metadata nền tảng runtime-agnostic trong `internal/skills`:
- `skills.Supported(deviceCaps)` cho skills, `skills.SupportedHooks(deviceCaps)`
  cho hooks, với `deviceCaps = device.Capabilities(config.DeviceTypeOrDefault())`.
- Đừng hardcode danh sách skill/hook theo backend — gate y như OpenClaw.

---

## Checklist cho backend mới

- [ ] Package `internal/<name>/`; `*Service` implement **toàn bộ** `AgentGateway`.
- [ ] Mọi stub đều `// no-op because …` hoặc `// TODO(<name>-…)` — không thân rỗng.
- [ ] `domain.AgentRuntime<Name>` + entry `AgentRuntimes`; `factory.go` case.
- [ ] `install.sh` + `install.go` (`//go:embed` + `runtimereg.Register`).
- [ ] **Setup có-state → `presync.sh`** (`//go:embed` + `runtimereg.RegisterPresync`),
      materialize bởi os-server mỗi switch. Không gì reset-fragile chỉ nằm trong
      `install.sh`.
- [ ] `verify` hook rẻ (CLI có mặt), không phải structure-check.
- [ ] `migrate_persona/openclaw_to_<name>.go` + chiều ngược: SOUL(+inline identity),
      MEMORY+daily+KNOWLEDGE (fold vào file backend LOAD THEO TÊN), USER;
      `Overwrite=true`; chiều ngược strip artifact riêng-backend.
- [ ] Skills: copy-import + **restore-trong-presync** (guard) + `skill_watcher.go`
      (song song openclaw, share `internal/skills/skillzip.go`).
- [ ] Hooks: native-backend hoặc OS-side — đã quyết & ghi (không thiếu âm thầm).
- [ ] `reset_<name>.go` + `factoryreset.go` case; **`agent_state.json` wipe cùng
      `config.json`**.
- [ ] Gate capability qua `skills.Supported` / `SupportedHooks`.
- [ ] Notify agent khi skill đổi qua `SendSystemChatMessage`.
- [ ] Docs: cập nhật backend doc kiểu `docs/hermes.md` + checklist này nếu hợp đồng đổi.

---

## Trạng thái parity Hermes (sổ trung thực)

**Xong:** nối switch, install nhúng + presync (config self-heal, restore skills),
migrate persona/memory (SOUL + inline identity, MEMORY + daily + KNOWLEDGE, USER),
`UpdateIdentityName`, skill watcher, factory-reset khoá-bước `agent_state.json`.

**Còn mở / no-op:** hooks (`emotion-acknowledge`, `turn-gate` — §6),
`WriteMCPEntry`/`RemoveMCPEntry` (`TODO(hermes-mcp)`), `CompactSession`,
`FetchChatHistory`, và nhóm model-sync (`StartModelSync`, `UpdatePrimaryModel`,
`StartPrimaryModelWatch`, `RefreshModelsConfig` — phần lớn N/A vì os-server gửi
model cố định tới custom provider campaign-api).
