# Validação visual de DPI e escala — Firebird Toolbox v1.2.31

A interface Win32 continua compacta e deve ser verificada em Windows real nas escalas abaixo antes da publicação final:

- 100% (96 DPI)
- 125% (120 DPI)
- 150% (144 DPI)
- 175% (168 DPI)

## Checklist

1. Cabeçalhos das seções 1 a 4 e Log de execução sem corte ou sobreposição, com o traço superior retomando logo após o título.
2. Título, versão e subtítulo sem colisão.
3. Primeira linha com Executar, Parar, Abrir pasta, Copiar caminho e Ajuda totalmente visíveis; segunda linha com Abrir relatório e Localizar banco sem truncamento.
4. Campos Firebird, porta, usuário e senha sem sobreposição.
5. Linha `Etapa` mostrando `Fase X/Y` sem truncar o texto principal.
6. Resultado legível em sucesso, parcial, erro e não apto.
7. Barra de progresso preenchendo visualmente até 100%.
8. Rodapé Status/crédito sem corte.
9. Janela Ajuda com botões inferiores visíveis.
10. Abrir relatório e Localizar banco devem ficar cinza quando indisponíveis e ganhar destaque azul assim que habilitados.
11. Abrir/fechar Ajuda e alterar rotina/versão sem deixar texto antigo na tela.

O sandbox Linux consegue validar dimensões declaradas, compilação e regressões estáticas, mas a inspeção final de renderização em DPI deve ser feita no Windows.
