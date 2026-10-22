# Media storage

Uploaded media (campaign images, attachments, logos) is stored either on the listnun server's disk or in S3-compatible object storage. Each workspace chooses in **Settings › Media uploads**.

## Local filesystem

Files go in the **upload path** on the server (default `uploads`, relative to the working directory) and are served from the **upload URI** (default `/uploads`). With Docker, mount that path on a volume so uploads survive container rebuilds. See [Configuration](../configuration.md#media-uploads).

The filesystem only works for a single server. Use S3-compatible storage if you run more than one, or host many workspaces.

## S3-compatible storage

Any service that speaks the S3 API works: AWS S3, Hetzner Object Storage, Cloudflare R2, Backblaze B2, MinIO, RustFS, Garage and others.

| Setting | Value |
| :------ | :---- |
| **Endpoint** | The provider's S3 URL, e.g. `https://fsn1.your-objectstorage.com` (Hetzner) or `https://s3.eu-west-1.amazonaws.com` (AWS). |
| **Region** | The bucket's region, e.g. `eu-central`, or `us-east-1` for most self-hosted stores. |
| **Bucket** | The bucket name. |
| **Bucket path** | A folder inside the bucket, or `/` for the root. |
| **Access key** / **Secret** | Credentials that can read and write the bucket. |
| **Bucket type** | **Public**: files get plain links. **Private**: links are signed and expire after the expiry you set. |
| **Public URL** | Optional, e.g. a CDN or custom domain in front of the bucket. Links use it instead of the endpoint. |

listnun uses path-style addresses (`endpoint/bucket/file`), which every S3-compatible store supports.

### Public buckets

For a public bucket, links in emails must open for anyone. listnun marks uploads `public-read`, but many S3-compatible stores (MinIO, RustFS and others) ignore per-file permissions. Give the bucket a policy that lets anyone read objects instead:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {"AWS": ["*"]},
    "Action": ["s3:GetObject"],
    "Resource": ["arn:aws:s3:::listnun-media/*"]
  }]
}
```

Only reading is public. Uploading and deleting still need the access key.

### Trying it locally with RustFS

RustFS is a small S3-compatible server, handy for development:

```shell
docker run -d --name listnun-s3 --restart unless-stopped \
  -p 127.0.0.1:9200:9000 -p 127.0.0.1:9201:9001 \
  -v listnun-s3-data:/data \
  -e RUSTFS_ACCESS_KEY=listnun -e RUSTFS_SECRET_KEY=listnun-secret \
  -e RUSTFS_CONSOLE_ENABLE=true \
  rustfs/rustfs:1.0.1
```

Create a bucket in the console at `http://127.0.0.1:9201` (or with any S3 client) and give it the policy above. Then in **Settings › Media uploads**, choose **S3-compatible** with endpoint `http://127.0.0.1:9200`, region `us-east-1`, your bucket, and the keys. Links will be local, so images won't load in emails opened on other devices.

## Storage use

listnun records each file's size. Workspaces see their total on the **Media** page, and the operator API reports it per workspace at `GET /api/operator/tenants/:id/storage`.
