# Terraform / Oracle Cloud (A0)

**A0 status: Ampere.** Always Free `VM.Standard.A1.Flex` is Running (1 OCPU / 6 GB). SSH: `uname -m` → `aarch64`. D3 Terraform lives in [`infra/`](../../infra/). D4 k3s migrate: [`k3s.md`](k3s.md).

| Field | Value |
|---|---|
| Outcome | Ampere |
| Date | 2026-10-10 |
| Tenancy home region | fill from console (home region is permanent) |
| Shape | `VM.Standard.A1.Flex`, 1 OCPU, 6 GB (2/12 was out of capacity) |
| Availability domain | not AD-2 (AD-2 returned out of host capacity); copy the AD from instance details |
| Instance OCID | copy from instance details |
| Public IPv4 | 137.131.14.30 (ephemeral; private 10.0.0.95) |
| Public hostname | https://shashank-dropdesk.duckdns.org |
| Notes | Ubuntu `ubuntu`. SSH wizard key `ssh-key-2026-10-10.key`. Keep Running. **D0:** Compose+Caddy on this VM ([deploy-compose-caddy.md](../runbooks/deploy-compose-caddy.md)). Postgres not published. NSG must allow 80/443. |

D3 Terraform-ifies this shape. D4 (Compose → k3s) runs because outcome is `Ampere`.

## What this is

deskdrop v2 production is one Always Free Oracle Cloud VM. Terraform owns VCN, NSG (22/80/443), the VM, DuckDNS, and Object Storage for state (`deskdrop-tfstate`) and Postgres backups (`deskdrop-backups`, [backups.md](backups.md)). A0 is the human gate: create the account and **claim the VM now**, before any app code.

Always Free Ampere A1 (as of this writing): **2 OCPU and 12 GB RAM total** per tenancy (`VM.Standard.A1.Flex`), home region only. That is enough for Compose and for k3s later. The AMD micro (`VM.Standard.E2.1.Micro`, 1 GB) is **not** a k3s host.

Stay on Always Free. Do not upgrade to Pay As You Go. Do not create paid shapes. The card at signup is for identity; it must not be charged.

## What D3 / D4 will do with each outcome

- **Ampere:** D0 = Compose + Caddy on this VM. D3 = import/manage this instance in Terraform. D4 = migrate live Compose to k3s on the same VM. Grafana Cloud or in-cluster.
- **micro only:** D0 = Compose + Caddy on the micro (API + Postgres + Caddy + dashboard only). D3 = Terraform that micro. D4 skipped. k3d-in-CI promoted into D1. Grafana Cloud only. Record k3s-on-Oracle as a miss, do not fake it on 1 GB.
- **refused:** no Oracle account. Next ₹0 host (GCP e2-micro, then home lab + DuckDNS, disclose it). D3 targets that host. k3s-on-Oracle is a miss. Do not pay.

## How to do A0

1. Open [Oracle Cloud Free Tier](https://www.oracle.com/cloud/free/) and start signup: [signup.cloud.oracle.com](https://signup.cloud.oracle.com/).
2. **Home region is permanent.** Always Free compute exists only in the home region. Prefer a region that still has Ampere capacity (often US: Phoenix `us-phoenix-1` or San Jose `us-sanjose-1`). Do not pick Chuncheon for A1.
3. Complete phone and card verification. Do not click Upgrade / Pay As You Go afterward.
4. Sign in to the [OCI Console](https://cloud.oracle.com/). Confirm you can open Compute → Instances.
5. Create a VCN if the console wizard has not (default VCN is fine for A0). Security list or NSG: allow SSH (22) from your IP for now; 80/443 can wait until D0.
6. Compute → Instances → Create instance:
   - Image: Ubuntu 24.04 (or Oracle Linux). Architecture **AArch64** for Ampere.
   - Shape: **`VM.Standard.A1.Flex`**, 2 OCPU, 12 GB memory. Look for Always Free-eligible.
   - Boot volume: default (~47 GB) is fine; stay within 200 GB Always Free block storage.
   - SSH key: paste **your** public key. Do not commit the private key.
   - Assign a public IPv4.
7. If create fails with **out of host capacity**: retry the other availability domain in the same region. Wait and retry. Try 1 OCPU / 6 GB once if 2/12 is refused (still record `Ampere` if A1 Flex starts). Do **not** upgrade the account for capacity.
8. If A1 never starts but `VM.Standard.E2.1.Micro` does: record `micro only`.
9. If signup itself is denied: record `refused`.

Idle Always Free instances can be reclaimed if CPU, network, and (for A1) memory stay under 20% for 7 days. After A0, leave a small idle-prevention note for D0 (a cron ping or real traffic from the beta). Do not keep an empty VM dark for a week.

## How to classify the outcome

- **Ampere:** an instance with shape `VM.Standard.A1.Flex` exists and is Running.
- **micro only:** account works, A1 never became Running, at least one `VM.Standard.E2.1.Micro` is Running (or you stopped after A1 failed and did not want the micro yet — still write `micro only` if the account exists but A1 is unavailable).
- **refused:** you cannot log into an OCI tenancy.

## Verify or demo (A0)

Replace placeholders after the instance exists.

```bash
# Console: you can sign in at https://cloud.oracle.com/

# From a machine with oci CLI configured (optional for A0; curl/ssh is enough):
# oci iam region-subscription list
# oci compute instance list --compartment-id "$COMPARTMENT_OCID" --output table

# SSH to the instance (Ampere or micro) using the key you uploaded:
ssh -i ~/.ssh/<your-oci-key> ubuntu@<PUBLIC_IP> 'uname -m && nproc && free -h'

# Ampere (AArch64) should print aarch64 and 2 CPUs (or 1 if you took 1 OCPU).
# Micro (x86_64) should print x86_64 and ~1 GB RAM.

# Copy these into the table at the top:
#   oci compute instance get --instance-id <INSTANCE_OCID> \
#     --query 'data.{id:id,shape:shape,ad:"availability-domain",state:"lifecycle-state"}'
```

If `uname -m` is `aarch64`, set outcome to `Ampere`. If `x86_64` and ~1 GB RAM, set outcome to `micro only`. If you never got a shell, set `refused` or keep retrying A1 in another AD **without** upgrading the account.

## D3 — Terraform of this VM

Code: [`infra/`](../../infra/). Shape is locked to **A1 Flex 1 OCPU / 6 GB**. Do not change shape. State is **not** in git (`*.tfstate` gitignored). The instance, VCN, and subnet use `prevent_destroy` and `ignore_changes = all` so a later `plan` cannot replace the live box.

**Never `terraform apply` on empty state.** That would try to create a second instance. Import first.

### 0. API key and tfvars

1. Console → Identity → API keys → add PEM (not the SSH key). Save it as `~/.oci/oci_api_key.pem` (the native `oci` backend and the provider both use this path).
2. Copy [`infra/terraform.tfvars.example`](../../infra/terraform.tfvars.example) → `infra/terraform.tfvars`.
3. Fill OCIDs from instance / VCN / subnet details. `public_ip` stays `137.131.14.30`. `duckdns_domain` is `shashank-dropdesk`. DuckDNS token from duckdns.org.

Terraform **>= 1.12** is required (`backend "oci"`). State is native OCI Object Storage, not the S3-compat API.

### 1. State bucket (once)

Console → Object Storage → Create bucket **`deskdrop-tfstate`** in the same compartment (Always Free). Enable versioning if the console offers it. Then:

```bash
cd infra
cp backend.hcl.example backend.hcl
# set namespace (oci os ns get), tenancy/user OCIDs, fingerprint, region
terraform init -backend-config=backend.hcl
```

[`infra/backend.tf`](../../infra/backend.tf) is already `backend "oci"` with `bucket = "deskdrop-tfstate"` and `private_key_path = "~/.oci/oci_api_key.pem"`. Until the bucket exists, rename `backend.tf` aside and `terraform init -backend=false` (local state still gitignored).

### 2. Import (required)

```bash
cd infra
terraform import oci_core_instance.deskdrop "$INSTANCE_OCID"
terraform import oci_core_vcn.deskdrop "$VCN_OCID"
terraform import oci_core_subnet.public "$SUBNET_OCID"
terraform plan
```

Accept: **0 to destroy**, **0 to replace**. Expect creates: NSG + 22/80/443 rules, `deskdrop-backups`, DuckDNS refresh. If `deskdrop-tfstate` already exists from step 1:

```bash
terraform import oci_objectstorage_bucket.tfstate "$NAMESPACE/deskdrop-tfstate"
```

If plan wants to replace the instance, stop. Do not apply.

### 3. Apply

```bash
terraform apply
```

Attach the new NSG `deskdrop` to the instance VNIC in the console (Terraform does not mutate the imported VNIC). Keep host iptables 80/443 as in [deploy-compose-caddy.md](../runbooks/deploy-compose-caddy.md).

### 4. Verify

```bash
terraform plan   # should settle near empty (DuckDNS may refresh)
curl -sfS https://shashank-dropdesk.duckdns.org/healthz
ssh -i ~/Downloads/ssh-key-2026-10-10.key ubuntu@137.131.14.30 'uname -m'
```

`uname -m` must still be `aarch64`. Public IP must still be 137.131.14.30.

## Failure modes (D3)

- Empty-state apply creates a second A1 VM (capacity / Always Free). Import first.
- Wrong image/source in config vs imported instance → replace. `ignore_changes = all` on instance/VCN/subnet is there to stop that.
- DuckDNS token in git: `terraform.tfvars` is gitignored; never commit it.
- Remote state 403 / init fail: `deskdrop-tfstate` missing, wrong `namespace` in `backend.hcl`, API key not at `~/.oci/oci_api_key.pem`, or Terraform older than 1.12 (native `oci` backend).
- NSG created but not on the VNIC: 80/443 still depend on the existing security list + iptables until you attach it.
