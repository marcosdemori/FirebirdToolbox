# Firebird Toolbox v1.2.8 — Ajuda rápida

Na v1.2.8, a barra de progresso sempre encerra visualmente em 100% quando a rotina termina. A cor e o campo `Resultado` indicam se o término foi completo, parcial, interrompido ou com erro.



AJUDA INTEGRADA
----------------
O botão Ajuda abre uma janela interna com guia rápido, descrição das rotinas, recuperação avançada, skip_data, estados de resultado, arquivos RECOVERY, erros comuns e boas práticas. A janela possui rolagem, Copiar ajuda e Fechar, sem depender de programa externo.


## Legibilidade da interface

Na v1.2.5, os rótulos de arquivos e instalações Firebird foram ampliados. A área inferior também foi reorganizada: `Etapa atual`, `Progresso` e `Resultado` aparecem em linhas separadas para que mensagens de recuperação e o resumo final não sejam cortados.

Na v1.2.8, o texto de ajuda do campo `Pular tabelas (regex)` foi reposicionado e ampliado para exibir integralmente `Regex opcional do gbak -skip_data (FB 3.0+)`, sem cortar o rótulo e sem aumentar a janela.

## Rotinas

A interface possui somente:

- `Corrigir banco`
- `Corrigir banco (avançado)`
- `Somente restaurar`

## Quando usar o modo avançado

Use `Corrigir banco (avançado)` quando:

- a aplicação informa banco corrompido;
- a validação normal falha;
- o `gbak` não consegue gerar backup;
- há suspeita de páginas, registros, BLOBs ou índices danificados;
- você precisa tentar salvar o máximo possível sem executar `mend` diretamente no original.

## Como funciona a recuperação avançada

1. O Toolbox verifica o Firebird selecionado.
2. Executa `gfix -validate -full -no_update` no original.
3. Consulta transações em limbo com `gfix -list`.
4. Cria uma pasta `RECOVERY_...` na pasta de backup.
5. Copia o banco original para essa pasta, calcula SHA-256, relê o original ao final e cancela se tamanho, data ou conteúdo mudarem durante a cópia.
6. Confere a cópia em modo somente leitura.
7. Se houver somente erro de índice, evita `mend`.
8. Se houver corrupção mais séria, pergunta antes de executar `gfix -mend -ignore` na cópia.
9. Gera o backup de recuperação.
10. Faz restore de teste automático.
11. Se o restore falhar por PRIMARY/UNIQUE duplicada, preserva o restore falho e refaz a restauração com `-inactive` para inspecionar todos os registros com os índices inativos.
12. Compara as ocorrências da chave; duplicidades idênticas podem ser saneadas na cópia, enquanto diferenças/inconclusões exigem confirmação explícita e ficam preservadas em `CONFLITOS_DUPLICIDADE.txt`.
13. `-one_at_a_time` fica reservado a outras falhas de restore que possam produzir um banco parcial útil para inspeção.
14. Se o restore de teste completo passar, pode restaurar o banco final.
15. Salva `RELATORIO_RECUPERACAO.txt`.


## Quando o backup ainda falha depois do mend

O Toolbox mantém uma saída técnica completa em `LOG_TECNICO_RECUPERACAO.txt`, mesmo com o painel em modo resumido. Se o `gbak` falhar, o Toolbox:

1. registra as mensagens relevantes da falha;
2. tenta apontar a relação provavelmente processada no momento do erro;
3. tenta criar um backup somente de metadados;
4. registra a comparação de corrupção antes/depois do `mend`;
5. inclui no relatório o trecho novo do `firebird.log`, se disponível;
6. se a falha apontar corrupção de índice, tenta desativar apenas índices de usuário não vinculados a constraints na cópia de trabalho e repete o backup sem excluir dados;
7. se ainda falhar, em Firebird 3.0+ pode criar um backup-base com `-skip_data` para a relação identificada;
8. restaura esse backup-base em um banco temporário e tenta recolocar os registros legíveis por leitura NATURAL;
9. se a leitura natural falhar e houver chave primária numérica simples, divide a faixa da chave progressivamente para isolar apenas os registros irrecuperáveis;
10. gera um novo backup com os registros recuperados e continua exigindo restore de teste.

Se todos os registros forem recuperados, o resultado volta a ser **RECUPERAÇÃO COMPLETA**. Se algumas chaves não puderem ser copiadas, o resultado é **RECUPERAÇÃO PARCIAL** e o relatório lista essas chaves.

## Pular tabelas irrecuperáveis

Em Firebird 3.0 ou superior, o campo `Pular tabelas (regex)` fica habilitado no modo avançado.

Exemplo:

```text
(CLIENTES_TEMP|HISTORICO_CORROMPIDO)
```

Quando você preenche esse campo manualmente, o Toolbox entende que deseja excluir os dados correspondentes do backup. Quando a tabela é identificada automaticamente durante uma falha, a v1.2.5 usa `-skip_data` apenas como etapa intermediária para construir um banco limpo e depois reinserir os registros legíveis.

## Resgate de registros da tabela problemática

Na recuperação automática, a v1.2.5 tenta preservar os itens da tabela que bloqueou o `gbak`:

- primeiro faz leitura `PLAN (T NATURAL)` para evitar índices danificados;
- copia os registros para um banco temporário restaurado sem os dados daquela relação;
- triggers ativos da relação são desativados somente durante a cópia e reativados antes do novo backup;
- índices que tenham sido temporariamente desativados são reconstruídos no banco temporário antes do backup final;
- se a leitura natural não terminar e houver chave primária numérica simples, o Toolbox isola falhas por faixas da chave;
- o relatório registra o número de registros recuperados e as chaves que não puderam ser copiadas.

## Chaves primárias/únicas duplicadas no restore

Quando o `gbak` informa `violation of PRIMARY or UNIQUE KEY constraint`, a v1.2.6:

- extrai automaticamente tabela, constraint e valores da chave problemática;
- preserva o restore originalmente falho;
- restaura novamente com `-inactive`, mantendo todos os índices/constraints inativos para inspeção;
- preserva uma cópia desse banco antes de qualquer exclusão;
- compara as ocorrências nas colunas compatíveis e grava `CONFLITOS_DUPLICIDADE.txt`;
- se as linhas forem idênticas nas colunas comparáveis, mantém uma ocorrência na cópia e remove apenas as extras;
- se houver campos diferentes ou BLOB/ARRAY que tornem a comparação inconclusiva, não escolhe silenciosamente: exige confirmação para manter a primeira ocorrência física e mantém todas as versões no banco de inspeção;
- reconstrói todos os índices, valida, gera `*_SEM_DUPLICIDADES_XX.BAK` e repete o restore completo;
- se o usuário autorizou descartar uma ocorrência diferente/inconclusiva, o resultado permanece **RECUPERAÇÃO PARCIAL**.

Arquivos adicionais: `*_RESTORE_FALHO_XX.RED`, `*_CONFLITO_ORIGINAL_XX.RED`, `*_CONFLITO_SANEADO_XX.RED`, `*_SEM_DUPLICIDADES_XX.BAK` e `CONFLITOS_DUPLICIDADE.txt`.

## Transações em limbo

O programa apenas detecta e relata transações em limbo. Ele não executa commit ou rollback automaticamente. Se elas impedirem o backup, pode ser oferecida uma tentativa com `gbak -limbo`.

## Restore de teste

Um backup só é marcado como verificado depois que o Toolbox consegue restaurá-lo de verdade em um arquivo temporário. Criar o `.BAK` sem erro não é suficiente no modo avançado.

## Resultado parcial

Se somente `-one_at_a_time` conseguir gerar um banco parcial, o programa:

- preserva o arquivo parcial;
- registra o caminho no relatório;
- mostra erro/aviso em vez de declarar sucesso completo.

## Se nem gfix/gbak resolverem

O Toolbox preserva o original, a cópia de trabalho e o relatório e encerra indicando necessidade de recuperação de baixo nível. Não tenta inventar reparos binários no arquivo do banco.

## Versões suportadas

`1.0` • `1.5` • `2.0` • `2.1` • `2.5` • `3.0` • `4.0` • `5.0`

Em versões antigas, o Toolbox também interpreta mensagens textuais do `gfix`, pois algumas versões podem retornar código de processo zero mesmo quando houve falha. Se o `gfix` legado não reconhecer `-no_update`, a validação completa é deslocada para a cópia de trabalho para preservar o original.

## Cuidados importantes

- Feche todos os sistemas conectados antes da recuperação.
- Garanta que a pasta escolhida para backup seja acessível ao serviço Firebird.
- Não apague o banco original depois da primeira aparente recuperação.
- Homologue o banco restaurado antes de substituir o banco de produção.
- `gfix -mend`, `gbak -ignore`, `-limbo`, `-skip_data` e `-one_at_a_time` podem resultar em perda ou ausência de dados; são recursos de resgate.
