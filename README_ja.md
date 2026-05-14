# Aruba Instant On Exporter

[English](README.md) | 日本語

Aruba Instant Onネットワークインフラ向けのPrometheusエクスポーターです。サイト、デバイス、接続されたクライアントの包括的な監視メトリクスを提供します。

## 機能

- **サイト監視**: サイトの健全性、ステータス、設定の追跡
- **デバイスメトリクス**: アクセスポイント、スイッチ、その他のネットワークデバイスの監視
- **クライアント分析**: 無線・有線クライアント接続の追跡
- **ネットワークパフォーマンス**: 信号品質、稼働時間、接続統計の監視
- **Prometheus統合**: 簡単な統合のためのネイティブPrometheusメトリクス形式

## エクスポートされるメトリクス

### サイトメトリクス

#### `aruba_instant_on_sites_total`
- **タイプ**: Gauge
- **説明**: 管理下にあるサイトの総数
- **ラベル**: なし

#### `aruba_instant_on_site_info`
- **タイプ**: Gauge
- **説明**: サイトの詳細情報（値は常に1）
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名
  - `health`: サイトの健全性ステータス（Good/Fair/Poor）
  - `status`: サイトのステータス（Up/Down）
  - `timezone`: サイトのタイムゾーン

#### `aruba_instant_on_site_throughput_bits_per_second`
- **タイプ**: Gauge
- **説明**: サイト全体の現在のスループット（bits per second）
- **ラベル**: `site_id`, `site_name`

#### `aruba_instant_on_site_data_transferred_24h_bytes`
- **タイプ**: Gauge
- **説明**: サイトで過去24時間に転送されたバイト総数
- **ラベル**: `site_id`, `site_name`

#### `aruba_instant_on_site_devices_up`
- **タイプ**: Gauge
- **説明**: サイトで稼働中のデバイス数（`aruba_instant_on_devices_total` と組み合わせて down 数を算出可能）
- **ラベル**: `site_id`, `site_name`

#### `aruba_instant_on_site_active_alerts`
- **タイプ**: Gauge
- **説明**: サイトの未解決アラート数（重要度別）
- **ラベル**: `site_id`, `site_name`, `severity`（`info` / `minor` / `major`）

#### `aruba_instant_on_site_networks_configured`
- **タイプ**: Gauge
- **説明**: サイトに構成されているネットワーク数
- **ラベル**: `site_id`, `site_name`, `kind`（`wired` / `wireless` / `total`）

#### `aruba_instant_on_site_networks_active`
- **タイプ**: Gauge
- **説明**: サイトで現在稼働中のネットワーク数
- **ラベル**: `site_id`, `site_name`, `kind`（`wired` / `wireless` / `total`）

### デバイスメトリクス

#### `aruba_instant_on_devices_total`
- **タイプ**: Gauge
- **説明**: サイトごとのデバイス総数
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名

#### `aruba_instant_on_device_info`
- **タイプ**: Gauge
- **説明**: デバイスの詳細情報（値は常に1）
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名
  - `device_id`: デバイスの一意識別子
  - `device_name`: デバイス名
  - `device_type`: デバイスタイプ（accessPoint/switch等）
  - `model`: デバイスモデル
  - `serial_number`: シリアル番号
  - `mac_address`: MACアドレス
  - `ip_address`: IPアドレス
  - `status`: デバイスステータス
  - `operational_state`: 運用状態

#### `aruba_instant_on_device_uptime_seconds`
- **タイプ**: Gauge
- **説明**: デバイスの稼働時間（秒）
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名
  - `device_id`: デバイスの一意識別子
  - `device_name`: デバイス名

### クライアントメトリクス

#### `aruba_instant_on_wireless_clients_total`
- **タイプ**: Gauge
- **説明**: サイトごとの無線クライアント総数
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名

#### `aruba_instant_on_wired_clients_total`
- **タイプ**: Gauge
- **説明**: サイトごとの有線クライアント総数
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名

#### `aruba_instant_on_clients_by_network`
- **タイプ**: Gauge
- **説明**: SSID別の現在の無線クライアント数（`networksSummary` 由来、重複排除済み）
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名
  - `network_ssid`: ネットワークSSID名

#### `aruba_instant_on_clients_by_network_band`
- **タイプ**: Gauge
- **説明**: SSID別の無線クライアント数を無線バンド別に分解
- **ラベル**: `site_id`, `site_name`, `network_ssid`, `band`（`2.4ghz` / `5ghz` / `6ghz`）

#### `aruba_instant_on_clients_by_ap`
- **タイプ**: Gauge
- **説明**: アクセスポイントごとのクライアント数
- **ラベル**:
  - `site_id`: サイトの一意識別子
  - `site_name`: サイト名
  - `device_id`: アクセスポイントのデバイスID
  - `device_name`: アクセスポイント名

### ネットワークメトリクス（SSID別）

#### `aruba_instant_on_network_throughput_bits_per_second`
- **タイプ**: Gauge
- **説明**: SSID別のスループット（bits per second、方向別）
- **ラベル**: `site_id`, `site_name`, `network_ssid`, `direction`（`upstream` / `downstream`）

#### `aruba_instant_on_network_health`
- **タイプ**: Gauge（info形式 — 報告された状態に対して値は常に `1`）
- **説明**: SSID別の健全性状態
- **ラベル**: `site_id`, `site_name`, `network_ssid`, `health`（`good` / `fair` / `problem` 等）

### アプリケーション利用メトリクス

#### `aruba_instant_on_app_category_data_transferred_24h_bytes`
- **タイプ**: Gauge
- **説明**: 過去24時間にネットワーク × アプリカテゴリごとに転送されたバイト数
- **ラベル**: `site_id`, `site_name`, `network`, `category`, `direction`（`upstream` / `downstream`）

### アラートメトリクス

#### `aruba_instant_on_alert_active`
- **タイプ**: Gauge（アラート発生中は `1`、解除されると系列が削除される）
- **説明**: 現在発生中（未解除）のアラート
- **ラベル**: `site_id`, `site_name`, `alert_id`, `type`（例: `deviceDown`）, `severity`

#### `aruba_instant_on_alert_age_seconds`
- **タイプ**: Gauge
- **説明**: 各アクティブアラートが発生してからの経過秒数
- **ラベル**: `site_id`, `site_name`, `alert_id`, `type`, `severity`

## インストール

### 前提条件

- Go 1.21以降
- 有効なAruba Instant Onアカウント認証情報
- Aruba Instant Onクラウドポータルへのアクセス

### ソースから

```bash
git clone https://github.com/csenet/instanton-exporter.git
cd instanton-exporter
go build -o instanton-exporter .
```

### Docker

GitHub Container Registryからプル:

```bash
docker pull ghcr.io/csenet/instanton-exporter:latest
```

環境変数で実行:

```bash
docker run -d \
  --name instanton-exporter \
  -p 9100:9100 \
  -e ARUBA_USERNAME=your-email@example.com \
  -e ARUBA_PASSWORD=your-password \
  ghcr.io/csenet/instanton-exporter:latest
```

docker-composeを使用:

```yaml
version: '3.8'
services:
  instanton-exporter:
    image: ghcr.io/csenet/instanton-exporter:latest
    ports:
      - "9100:9100"
    environment:
      - ARUBA_USERNAME=your-email@example.com
      - ARUBA_PASSWORD=your-password
    restart: unless-stopped
```

## 設定

### 環境変数

エクスポーターには以下の環境変数が必要です：

- `ARUBA_USERNAME` - Aruba Instant Onアカウントのメールアドレス
- `ARUBA_PASSWORD` - Aruba Instant Onアカウントのパスワード

### .envファイルの使用

プロジェクトディレクトリに`.env`ファイルを作成：

```env
ARUBA_USERNAME=user@example.com
ARUBA_PASSWORD=your-secure-password
```

### コマンドライン

```bash
export ARUBA_USERNAME="user@example.com"
export ARUBA_PASSWORD="your-secure-password"
./instanton-exporter
```

## 使用方法

1. 認証情報を設定（設定セクションを参照）
2. エクスポーターを実行：
   ```bash
   ./instanton-exporter
   ```
3. エクスポーターはデフォルトでポート`9100`で起動
4. メトリクスは`http://localhost:9100/metrics`で利用可能

### サンプル出力

```
# HELP aruba_instant_on_sites_total Total number of sites
# TYPE aruba_instant_on_sites_total gauge
aruba_instant_on_sites_total 2

# HELP aruba_instant_on_device_uptime_seconds Device uptime in seconds
# TYPE aruba_instant_on_device_uptime_seconds gauge
aruba_instant_on_device_uptime_seconds{device_id="...",device_name="Office-AP",site_id="...",site_name="Main Office"} 86400

# HELP aruba_instant_on_wireless_clients_total Total number of wireless clients
# TYPE aruba_instant_on_wireless_clients_total gauge
aruba_instant_on_wireless_clients_total{site_id="...",site_name="Main Office"} 15
```

## Prometheus設定

`prometheus.yml`に以下を追加：

```yaml
scrape_configs:
  - job_name: 'aruba-instant-on'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 30s
    scrape_timeout: 10s
```

## Grafanaダッシュボード

メトリクスはGrafanaを使用して可視化できます。主要なダッシュボードパネルには以下が含まれます：

- サイトヘルス概要
- デバイスステータスと稼働時間
- クライアント接続トレンド
- SSIDごとのネットワーク使用率
- アクセスポイントパフォーマンス

## 認証

エクスポーターは安全な認証のためにArubaのOAuth 2.0とPKCE（Proof Key for Code Exchange）を使用します：

1. Arubaポータルから動的設定を取得
2. セッショントークンを取得するためのMFA検証を実行
3. PKCEを使用してセッショントークンを認証コードに交換
4. API呼び出し用のアクセストークンを取得

## APIレート制限

エクスポーターはデフォルトで30秒ごとにメトリクスを収集します。Aruba Instant On APIにはレート制限があるため、収集間隔を過度に短く設定することは避けてください。

## 開発

### プロジェクト構造

```
├── main.go              # メインアプリケーションとPrometheusメトリクス
├── auth/                # 認証処理
│   ├── client.go        # OAuth2/PKCE認証
│   └── pkce.go          # PKCE実装
├── models/              # データ構造
│   └── settings.go      # 設定モデル
├── go.mod               # Goモジュール定義
└── .env.example         # 環境設定テンプレート
```

### ビルド

```bash
go mod tidy
go build -o instanton-exporter .
```

### テスト

```bash
go test ./...
```

## セキュリティ考慮事項

- 認証情報を安全に保存（環境変数、シークレット管理）
- エクスポーターはすべてのAPI通信でHTTPSを使用
- OAuth2トークンは必要に応じて自動的に更新
- 認証情報はログに記録されず、メトリクスに公開されません

## トラブルシューティング

### 認証問題

- Aruba Instant On認証情報を確認
- アカウントが監視したいサイトへのアクセス権を持っていることを確認
- アカウントのMFA要件をチェック

### ネットワーク問題

- `portal.instant-on.hpe.com`への発信HTTPS アクセスを確認
- エクスポーターのリスニングポート（9100）のファイアウォールルールをチェック

### デバッグ

ソースコード内のデバッグステートメントのコメントを外して再ビルドすることで、デバッグログを有効にできます。

## 貢献

1. リポジトリをフォーク
2. 機能ブランチを作成（`git checkout -b feature/amazing-feature`）
3. 変更をコミット（`git commit -m 'Add amazing feature'`）
4. ブランチにプッシュ（`git push origin feature/amazing-feature`）
5. プルリクエストを開く

## ライセンス

このプロジェクトはMITライセンスの下でライセンスされています - 詳細は[LICENSE](LICENSE)ファイルを参照してください。

## 免責事項

これは非公式のエクスポーターであり、HPE/Arubaと提携していません。自己責任でご使用ください。

## サポート

- バグレポートや機能リクエストについてはissueを作成
- 新しいissueを作成する前に既存のissueをチェック
- 問題を報告する際はログと設定詳細を含めてください

---

ネットワーク監視コミュニティのために ❤️ で作成
