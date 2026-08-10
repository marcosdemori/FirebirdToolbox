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
