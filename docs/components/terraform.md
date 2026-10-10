# Terraform / Oracle Cloud (A0)

**A0 status: pending.** Oracle signup has not succeeded yet. Work continues on later steps. When the account exists and an instance attempt has a result, replace `pending` with exactly one of: `Ampere` / `micro only` / `refused`.

| Field | Value |
|---|---|
| Outcome | pending |
| Date | |
| Tenancy home region | |
| Shape | |
| Availability domain | |
| Instance OCID | |
| Public IPv4 | |
| Notes | A0 blocked on OCI signup. Do not run Terraform until D3. |

A0 is not green while outcome is `pending`. D3 will Terraform-ify the shape recorded here. D4 (Compose → k3s) runs only if outcome is `Ampere`.

## What this is

deskdrop v2 production is one Always Free Oracle Cloud VM. Terraform (D3) will own VCN, NSG (22/80/443), the VM, DuckDNS, and Object Storage for state and Postgres backups. A0 is the human gate: create the account and **claim the VM now**, before any app code.

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

D3 will add `terraform init` / `plan` / `apply` / `import` here. Do not run Terraform in A0.
