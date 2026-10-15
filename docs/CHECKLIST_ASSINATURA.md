# Checklist — assinatura oficial do Firebird Toolbox

Antes de criar `v1.2.32`:

- [ ] Criar/validar conta Microsoft Artifact Signing no Azure.
- [ ] Criar Certificate Profile de assinatura de código.
- [ ] Criar App Registration/service principal para GitHub Actions.
- [ ] Criar credencial federada OIDC para `marcosdemori/FirebirdToolbox`.
- [ ] Atribuir a função `Artifact Signing Certificate Profile Signer`.
- [ ] GitHub Secret `AZURE_CLIENT_ID`.
- [ ] GitHub Secret `AZURE_TENANT_ID`.
- [ ] GitHub Secret `AZURE_SUBSCRIPTION_ID`.
- [ ] GitHub Variable `ARTIFACT_SIGNING_ENDPOINT`.
- [ ] GitHub Variable `ARTIFACT_SIGNING_ACCOUNT_NAME`.
- [ ] GitHub Variable `ARTIFACT_SIGNING_CERTIFICATE_PROFILE_NAME`.
- [ ] Push em `main` com CI verde.
- [ ] Só então criar/push da tag `v1.2.32`.

A Release falha antes de compilar se qualquer item de configuração estiver ausente.
