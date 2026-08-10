# Distribuição Windows, assinatura e Microsoft Defender

## Objetivo da v1.2.11

A v1.2.11 separa a lógica do Firebird do processo de distribuição do executável. O build de distribuição passa a gerar recursos nativos do Windows (ícone, manifesto e VersionInfo), validar esses dados e oferecer assinatura Authenticode antes de calcular os hashes e publicar a Release.

## Metadados esperados

Em **Propriedades > Detalhes** do EXE gerado pelo build Windows:

- Produto: `Firebird Toolbox`
- Empresa/desenvolvedor: `Marcos De Mori Laiola`
- Descrição: `Firebird Toolbox - manutenção e recuperação Firebird`
- Versão: `1.2.11`
- Copyright: `© 2026 Marcos De Mori Laiola. Todos os direitos reservados.`

Os recursos são definidos em `winres/winres.json` e gerados por `go-winres v0.3.3`.

## Build local

```powershell
.\tools\build.ps1 -Version "1.2.11"
```

Na primeira execução, se `go-winres` não existir no PATH, o script instala a versão fixada `v0.3.3`.

## Build local assinado

```powershell
.\tools\build.ps1 `
  -Version "1.2.11" `
  -SigningPfx "C:\Certificados\codigo.pfx" `
  -SigningPassword "SENHA_DO_CERTIFICADO"
```

A senha não deve ser gravada em scripts versionados.

## Assinatura no GitHub Actions

Configure os secrets do repositório:

- `WINDOWS_SIGN_CERT_BASE64`
- `WINDOWS_SIGN_CERT_PASSWORD`

O workflow `.github/workflows/release.yml`:

1. valida código e testes;
2. gera VersionInfo/manifesto/ícone;
3. compila x64 e x86 sem `-s -w`;
4. aplica Authenticode e timestamp se o certificado estiver configurado;
5. verifica a assinatura;
6. gera ZIP e SHA-256 somente depois da assinatura;
7. publica `SIGNATURE_STATUS.txt` na Release.

## Quando o Defender detectar um build

Não trate automaticamente o alerta como falso positivo e não recomende exclusão permanente de uma pasta inteira. Registre:

- versão;
- SHA-256;
- nome da detecção;
- status da assinatura;
- origem do download;
- print e horário da detecção.

Para um binário oficial que você verificou e acredita ter sido classificado incorretamente, envie-o para análise da Microsoft:

https://www.microsoft.com/en-us/wdsi/filesubmission

## Ferramentas oficiais

Documentação do SignTool:

https://learn.microsoft.com/windows/win32/seccrypto/signtool

Timestamp Authenticode:

https://learn.microsoft.com/windows/win32/seccrypto/time-stamping-authenticode-signatures
