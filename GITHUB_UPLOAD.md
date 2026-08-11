# Atualizar o Firebird Toolbox para v1.2.31

## 1. Substitua os arquivos do repositório

Copie a árvore desta versão sobre o repositório local e execute:

```powershell
git add .
git status
git commit -m "fix(ui): manter divisorias acima dos titulos v1.2.31"
git push origin main
```

Espere o workflow **CI** ficar verde.

## 2. Configure a assinatura ANTES de criar a tag

A v1.2.31 não publica Release sem Microsoft Artifact Signing.

No GitHub, em **Settings > Secrets and variables > Actions**, configure:

### Secrets

- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`
- `AZURE_SUBSCRIPTION_ID`

### Variables

- `ARTIFACT_SIGNING_ENDPOINT`
- `ARTIFACT_SIGNING_ACCOUNT_NAME`
- `ARTIFACT_SIGNING_CERTIFICATE_PROFILE_NAME`

No Azure, a identidade federada do GitHub deve possuir **Artifact Signing Certificate Profile Signer**.

Consulte `docs/DISTRIBUICAO_WINDOWS.md` para o passo a passo e os motivos de segurança.

## 3. Publique a tag somente depois da configuração

```powershell
git tag -a v1.2.31 -m "Firebird Toolbox v1.2.31"
git push origin v1.2.31
```

A Release só será criada se:

- build/testes passarem;
- os EXEs forem assinados;
- Authenticode estiver `Valid`;
- houver timestamp;
- os hashes forem gerados depois da assinatura.

Também serão geradas atestações de proveniência do GitHub.

## Repositório novo

Se ainda não houver repositório:

```powershell
git init
git branch -M main
git add .
git status
git commit -m "fix(ui): manter divisorias acima dos titulos v1.2.31"
git remote add origin https://github.com/SEU_USUARIO/FirebirdToolbox.git
git push -u origin main
```

Depois configure Artifact Signing e só então crie a tag.
