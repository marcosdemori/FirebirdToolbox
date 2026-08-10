# Contribuindo

Este repositório usa Go e tem interface Win32 nativa. O código principal possui
`//go:build windows`, então a validação completa deve ser feita para Windows.

## Antes de enviar uma alteração

1. Execute `gofmt` nos arquivos Go alterados.
2. Execute `go vet` para Windows.
3. Compile os testes para Windows x64 e x86.
4. Compile os executáveis x64 e x86.
5. Não inclua bancos, backups, logs reais, senhas ou dados de clientes.
6. Preserve as três rotinas principais da interface salvo mudança deliberada:
   `Corrigir banco`, `Corrigir banco (avançado)` e `Somente restaurar`.

No Windows, você pode executar:

```powershell
.\tools\build.ps1
```
