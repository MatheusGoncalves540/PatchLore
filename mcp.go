package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// O servidor MCP expõe o patchlore como ferramenta para agentes (Zed,
// Claude e afins). O protocolo é JSON-RPC, uma mensagem por linha no stdio;
// são poucos métodos, então a biblioteca padrão dá conta sem SDK.

type pedidoRPC struct {
	ID     json.RawMessage `json:"id"`
	Metodo string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type erroRPC struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var ferramenta = map[string]any{
	"name": "gerar_patchnotes",
	"description": "Gera patchnotes visuais (PNG, PDF, Markdown e notes.json) a partir dos commits entre duas refs de um repositório git local. " +
		"Sem 'notas', a IA configurada no patchlore escreve o texto; com 'notas', usa o texto dado e não chama IA nenhuma. " +
		"Sem antiga/nova, reaproveita as refs salvas no .patchlore.toml do repo.",
	"inputSchema": map[string]any{
		"type":     "object",
		"required": []string{"repositorio"},
		"properties": map[string]any{
			"repositorio": map[string]any{"type": "string", "description": "Caminho absoluto de uma pasta dentro do repositório git"},
			"antiga":      map[string]any{"type": "string", "description": "Ref de partida (o que já está publicado), ex.: production ou um commit^1"},
			"nova":        map[string]any{"type": "string", "description": "Ref de chegada (o que entra agora), ex.: development"},
			"titulo":      map[string]any{"type": "string", "description": "Título do rodapé e contexto para a IA, ex.: 'PR #10 · Novas skins'"},
			"saida":       map[string]any{"type": "string", "description": "Pasta de saída (padrão: patchlore-out na raiz do repo)"},
			"notas": map[string]any{
				"type":        "object",
				"description": "Texto pronto: {titulo, itens:[{tag, titulo, texto}]}, 1 a 6 itens; tags do tema padrão: novo, melhoria, correcao, regra",
			},
		},
	},
}

func servirMCP(entrada io.Reader, saida io.Writer) error {
	leitor := bufio.NewScanner(entrada)
	leitor.Buffer(make([]byte, 1024*1024), 16*1024*1024) // notas grandes cabem numa linha
	escritor := json.NewEncoder(saida)
	for leitor.Scan() {
		var p pedidoRPC
		if err := json.Unmarshal(leitor.Bytes(), &p); err != nil || len(p.ID) == 0 {
			continue // notificações (sem id) não pedem resposta
		}
		resultado, erro := atender(p)
		resposta := map[string]any{"jsonrpc": "2.0", "id": p.ID}
		if erro != nil {
			resposta["error"] = erro
		} else {
			resposta["result"] = resultado
		}
		if err := escritor.Encode(resposta); err != nil {
			return err
		}
	}
	return leitor.Err()
}

func atender(p pedidoRPC) (any, *erroRPC) {
	switch p.Metodo {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		json.Unmarshal(p.Params, &params)
		if params.ProtocolVersion == "" {
			params.ProtocolVersion = "2025-06-18"
		}
		return map[string]any{
			"protocolVersion": params.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "patchlore", "version": "0.2.0"},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": []any{ferramenta}}, nil
	case "tools/call":
		return chamarFerramenta(p.Params), nil
	}
	return nil, &erroRPC{Code: -32601, Message: "método desconhecido: " + p.Metodo}
}

// Erros da ferramenta voltam como resultado com isError, não como erro de
// protocolo: assim o agente lê a mensagem e consegue corrigir a chamada.
func chamarFerramenta(bruto json.RawMessage) map[string]any {
	var chamada struct {
		Nome string `json:"name"`
		Args struct {
			Repositorio string          `json:"repositorio"`
			Antiga      string          `json:"antiga"`
			Nova        string          `json:"nova"`
			Titulo      string          `json:"titulo"`
			Saida       string          `json:"saida"`
			Notas       json.RawMessage `json:"notas"`
		} `json:"arguments"`
	}
	texto, err := func() (string, error) {
		if err := json.Unmarshal(bruto, &chamada); err != nil {
			return "", err
		}
		if chamada.Nome != "gerar_patchnotes" {
			return "", fmt.Errorf("ferramenta desconhecida: %s", chamada.Nome)
		}
		a := chamada.Args
		if (a.Antiga == "") != (a.Nova == "") {
			return "", fmt.Errorf("informe antiga e nova juntas, ou nenhuma das duas")
		}
		ref := ""
		if a.Antiga != "" {
			ref = a.Antiga + ".." + a.Nova
		}
		o := opcoes{titulo: a.Titulo, saida: a.Saida}
		if len(a.Notas) > 0 && string(a.Notas) != "null" {
			o.notasProntas = a.Notas
		}
		// O processo do editor roda em qualquer pasta; a saída relativa vai
		// para dentro do repo, onde a pessoa espera encontrar.
		if o.saida == "" {
			o.saida = "patchlore-out"
		}
		if !filepath.IsAbs(o.saida) {
			o.saida = filepath.Join(a.Repositorio, o.saida)
		}
		var log bytes.Buffer
		if err := gerar(context.Background(), a.Repositorio, ref, o, &log); err != nil {
			return log.String(), err
		}
		md, _ := os.ReadFile(filepath.Join(o.saida, "patchnotes.md"))
		return fmt.Sprintf("%s\nArquivos em %s: notes.json, patchnotes.png, patchnotes.pdf, patchnotes.md\n\n%s", log.String(), o.saida, md), nil
	}()
	if err != nil {
		texto += "\nErro: " + err.Error()
	}
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": texto}},
		"isError": err != nil,
	}
}
