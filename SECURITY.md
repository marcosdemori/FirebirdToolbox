# Segurança

## Dados sensíveis

Nunca envie para Issues, Pull Requests ou commits:

- bancos reais de clientes (`.fdb`, `.gdb`, `.red`);
- backups (`.bak`, `.fbk`, `.nbk`);
- senhas SYSDBA;
- arquivos `.env`, certificados ou chaves;
- relatórios de recuperação contendo dados comerciais sensíveis.

O `.gitignore` do projeto bloqueia os formatos mais comuns, mas confira sempre
`git status` antes de fazer commit.

## Recuperação de bancos

As rotinas avançadas podem lidar com corrupção severa. Execute-as em cópias,
nunca sobre o único arquivo disponível. O Firebird Toolbox cria cópias de
trabalho quando aplicável, porém a responsabilidade pelo backup externo e pela
homologação do banco recuperado continua sendo do operador.

## Reportar uma falha de segurança

Não publique credenciais ou bancos de clientes em Issues públicas. Entre em
contato diretamente com o responsável pelo projeto para compartilhar detalhes
sensíveis por um canal privado.

## Microsoft Defender e distribuição dos executáveis

O Firebird Toolbox executa ferramentas administrativas do Firebird (`gfix`, `gbak`, `gstat` e `isql`) e manipula arquivos de banco, portanto builds novos e sem reputação podem receber classificação heurística de antivírus. Uma detecção deve ser investigada; não trate automaticamente qualquer alerta como falso positivo.

Antes de liberar um executável:

1. confira o SHA-256 publicado na Release;
2. confira em **Propriedades > Detalhes** se o produto é `Firebird Toolbox`, o desenvolvedor é `Marcos De Mori Laiola` e a versão corresponde à Release;
3. quando a Release estiver assinada, confira **Propriedades > Assinaturas Digitais** e exija assinatura válida;
4. não recomende excluir pastas inteiras do Microsoft Defender como solução permanente;
5. se um build oficial for classificado incorretamente, envie o arquivo à Microsoft para análise de falso positivo e mantenha o hash e o resultado da submissão junto do registro da Release.

## Assinatura Authenticode

O workflow de Release suporta assinatura opcional usando um certificado PFX. Configure estes secrets no repositório:

- `WINDOWS_SIGN_CERT_BASE64`: conteúdo Base64 do `.pfx`;
- `WINDOWS_SIGN_CERT_PASSWORD`: senha do certificado.

A chave privada não deve entrar no Git. Quando os secrets não existem, a Release ainda é gerada, mas inclui `SIGNATURE_STATUS.txt` marcando explicitamente `UNSIGNED`.
