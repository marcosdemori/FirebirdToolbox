# Distribuição Windows, assinatura e Microsoft Defender

## Objetivo da v1.2.12

> **v1.2.24:** mantém o executável único x86/x64 e a descoberta portátil do Firebird; preserva a abertura da pasta de resultado pelo shell nativo do Windows e melhora a Ajuda e os estados visuais. A lógica de recuperação avançada permanece.


A v1.2.12 endurece o executável e a cadeia de distribuição em dois níveis:

1. o aplicativo deixa de se autoelevar com `runas` e deixa de extrair ícones para `%TEMP%`; os ícones incorporados são carregados diretamente da memória;
2. uma Release oficial **não pode mais ser publicada sem assinatura digital**. O GitHub Actions exige Microsoft Artifact Signing, timestamp RFC3161 e verificação Authenticode antes do ZIP, dos hashes e da Release.

Isso reduz comportamentos que podem parecer suspeitos para mecanismos heurísticos e impede que um EXE oficial sem identidade criptográfica seja distribuído por engano.

> Nenhuma alteração de assinatura consegue prometer que um antivírus nunca produzirá falso positivo. A v1.2.12 garante que a distribuição oficial use a cadeia correta: build rastreável, assinatura confiável, timestamp, SHA-256, atestação de proveniência e preflight do Microsoft Defender quando o runner oferece os cmdlets.

## Metadados esperados

Em **Propriedades > Detalhes** do EXE oficial:

- Produto: `Firebird Toolbox`
- Empresa/desenvolvedor: `Marcos De Mori Laiola`
- Descrição: `Firebird Toolbox - manutenção e recuperação Firebird`
- Versão atual: `1.2.24`
- Copyright: `© 2026 Marcos De Mori Laiola. Todos os direitos reservados.`

Os recursos são definidos em `winres/winres.json` e gerados por `go-winres v0.3.3`.

## Configuração obrigatória do Microsoft Artifact Signing

A Release oficial usa autenticação OIDC entre GitHub Actions e Azure. Não há certificado PFX/Base64 armazenado no repositório.

### 1. No Azure

Crie/tenha:

- uma conta do **Artifact Signing**;
- um **Certificate Profile** de assinatura de código;
- uma App Registration / service principal para o GitHub Actions;
- uma credencial federada OIDC para este repositório;
- a função **Artifact Signing Certificate Profile Signer** atribuída à identidade usada pelo workflow.

### 2. GitHub Secrets

Em **Settings > Secrets and variables > Actions > Secrets**, configure:

- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`
- `AZURE_SUBSCRIPTION_ID`

Não é necessário `AZURE_CLIENT_SECRET` quando OIDC está configurado corretamente.

### 3. GitHub Variables

Em **Settings > Secrets and variables > Actions > Variables**, configure:

- `ARTIFACT_SIGNING_ENDPOINT` — exemplo regional: `https://eus.codesigning.azure.net/`
- `ARTIFACT_SIGNING_ACCOUNT_NAME`
- `ARTIFACT_SIGNING_CERTIFICATE_PROFILE_NAME`

O endpoint deve corresponder à região da conta do Artifact Signing.

## O que o workflow de Release faz

`.github/workflows/release.yml` executa, nesta ordem:

1. confere se a tag corresponde ao `appVersion`;
2. bloqueia a Release se qualquer configuração do Artifact Signing estiver ausente;
3. roda `gofmt`, `go vet` e `go test`;
4. gera VersionInfo, manifesto e ícone nativos;
5. compila um único PE32/x86 para Windows x86/x64;
6. autentica no Azure via OIDC (`azure/login`);
7. assina os dois EXEs com `azure/artifact-signing-action`;
8. aplica timestamp RFC3161 da Microsoft;
9. exige `Get-AuthenticodeSignature` com status `Valid` e timestamp presente;
10. executa um preflight com Microsoft Defender quando os cmdlets Defender estão disponíveis no runner;
11. cria ZIP e SHA-256 somente depois da assinatura;
12. gera atestações de proveniência do GitHub para o executável único e o ZIP;
13. cria a GitHub Release.

**Não existe fallback para Release sem assinatura.** Se o Artifact Signing não estiver configurado ou falhar, a Release falha e nenhum EXE é publicado.

## Build local

Build local sem assinatura é permitido apenas de forma explícita para desenvolvimento:

```powershell
.\tools\build.ps1 -Version "1.2.24" -AllowUnsignedDev
```

Os arquivos recebem `_UNSIGNED_DEV` no nome e `SIGNATURE_STATUS.txt` informa **NAO DISTRIBUIR**.

Para um build local assinado com um certificado PFX confiável:

```powershell
.\tools\build.ps1 `
  -Version "1.2.24" `
  -SigningPfx "C:\Certificados\codigo.pfx" `
  -SigningPassword "SENHA_DO_CERTIFICADO"
```

O PFX é apenas uma alternativa local. As Releases do GitHub usam Microsoft Artifact Signing.

## Mudanças no comportamento do aplicativo

### Sem autoelevação automática

O Toolbox não chama mais `runas` ao abrir. Ele inicia com o token normal do usuário (`asInvoker`). Isso segue privilégio mínimo e remove um padrão de autoelevação que pode parecer suspeito para heurísticas.

Se uma pasta ou operação específica exigir direitos administrativos, o Windows/Firebird retornará erro de permissão e o operador poderá abrir o Toolbox manualmente como Administrador somente naquele cenário.

### Ícones sem extração para TEMP

Os `.ico` continuam incorporados ao executável, mas não são mais gravados em `%TEMP%`. O Toolbox lê o ICO incorporado e cria `HICON` diretamente em memória com a API do Windows.

## Conferir uma Release oficial

No PowerShell:

```powershell
Get-AuthenticodeSignature ".\FirebirdToolbox_v1.2.24_x86_x64.exe" |
  Format-List Status,StatusMessage,SignerCertificate,TimeStamperCertificate
```

A Release oficial deve mostrar `Status : Valid` e um certificado de timestamp.

Confira também o SHA-256 publicado:

```powershell
Get-FileHash ".\FirebirdToolbox_v1.2.24_x86_x64.exe" -Algorithm SHA256
```

E, para repositório público, a proveniência do GitHub pode ser verificada com:

```powershell
gh attestation verify .\FirebirdToolbox_v1.2.24_x86_x64.exe -R marcosdemori/FirebirdToolbox
```

## Quando o Defender ainda detectar um build oficial assinado

Não crie exclusão permanente da pasta. Registre versão, SHA-256, nome da detecção e assinatura e envie **o EXE oficial assinado da Release** à Microsoft para análise de falso positivo:

https://www.microsoft.com/en-us/wdsi/filesubmission

Uma classificação de terceiros é externa ao código e pode mudar com definições/reputação; por isso a resposta correta é assinatura confiável + submissão do hash/binário oficial, e não enfraquecer o Defender.
