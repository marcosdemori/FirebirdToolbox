# Contribuindo

Este repositório usa Go e tem interface Win32 nativa. O código principal possui
`//go:build windows`, então a validação completa deve ser feita para Windows.

## Antes de enviar uma alteração

1. Execute `gofmt` nos arquivos Go alterados.
2. Execute `go vet` para Windows.
3. Compile/teste o código para Windows x64 e x86.
4. Compile o executável único de distribuição com `GOARCH=386` e valide-o como x86/x64.
5. Não inclua bancos, backups, logs reais, senhas ou dados de clientes.
6. Preserve as três rotinas principais da interface salvo mudança deliberada:
   `Manutenção / conversão`, `Recuperar banco corrompido` e `Somente restaurar`.

No Windows, você pode executar:

```powershell
.\tools\build.ps1
```
