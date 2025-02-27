package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/openai/openai-go"
	"github.com/rs/zerolog/log"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Узнаёт длительность файла через ffprobe
func getDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, err
	}
	str := strings.TrimSpace(out.String())
	return strconv.ParseFloat(str, 64)
}

// Делит аудиофайл на части через ffmpeg
func splitAudioFile(inputPath string, outputPattern string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-f", "segment",
		"-segment_time", "600", "-c", "copy", outputPattern)
	return cmd.Run()
}

// Отправляет файл в Whisper и возвращает JSON с сегментами
func transcribeAudio(ctx context.Context, client *openai.Client, filePath string, offset float64) ([]Segment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка при открытии файла: %v", err)
	}
	defer f.Close()

	resp, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		File:           openai.F[io.Reader](f),
		Model:          openai.F(openai.AudioModelWhisper1),
		Language:       openai.F("ru"),
		ResponseFormat: openai.F(openai.AudioResponseFormatVerboseJSON),
		TimestampGranularities: openai.F([]openai.AudioTranscriptionNewParamsTimestampGranularity{
			openai.AudioTranscriptionNewParamsTimestampGranularitySegment,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка при отправке запроса: %v", err)
	}

	// Парсим полученный JSON, там должна быть структура с массивом сегментов
	var result struct {
		Segments []Segment `json:"segments"`
	}
	err = json.Unmarshal([]byte(resp.Text), &result)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить JSON: %v", err)
	}

	// Смещаем таймкоды
	for i := range result.Segments {
		result.Segments[i].Start += offset
		result.Segments[i].End += offset
	}
	return result.Segments, nil
}

func main() {
	inputFile := `./whisper/audio/2025_02_15_диспут_Чегон_нельзя_делать_с_Мифом.m4a`
	outputPattern := `./whisper/audio/chunk/chunk_%03d.m4a`

	fmt.Println("Разделение аудиофайла на части...")
	err := splitAudioFile(inputFile, outputPattern)
	if err != nil {
		fmt.Printf("Ошибка при разделении файла: %v\n", err)
		return
	}

	// Инициализация клиента на основе переменной окружения
	client := openai.NewClient()

	var allSegments []Segment
	offset := 0.0

	for i := 0; ; i++ {
		chunkFile := fmt.Sprintf(`./whisper/audio/chunk/chunk_%03d.m4a`, i)
		if _, err := os.Stat(chunkFile); os.IsNotExist(err) {
			break
		}

		// Узнаём длительность текущего куска
		dur, err := getDuration(chunkFile)
		if err != nil {
			fmt.Printf("Не удалось получить длительность файла %s: %v\n", chunkFile, err)
			break
		}

		ctx := context.Background()
		segments, err := transcribeAudio(ctx, client, chunkFile, offset)
		if err != nil {
			log.Fatal().Err(err).Msgf("Ошибка при распознавании: %s", chunkFile)
		}
		allSegments = append(allSegments, segments...)

		// Увеличиваем смещение на длительность k-го куска
		offset += dur
	}

	// Сериализация всех сегментов в единый JSON
	data, err := json.MarshalIndent(allSegments, "", "  ")
	if err != nil {
		fmt.Printf("Ошибка при сериализации JSON: %v\n", err)
		return
	}
	if err := ioutil.WriteFile(`combined.json`, data, 0644); err != nil {
		fmt.Printf("Ошибка при сохранении файла combined.json: %v\n", err)
		return
	}
	fmt.Println("Объединённый JSON сохранён в файл `combined.json`")

	// Дополнительно собираем общий текст
	var textResult strings.Builder
	for _, seg := range allSegments {
		textResult.WriteString(seg.Text)
		textResult.WriteString("\n")
	}
	err = ioutil.WriteFile(`transcription.txt`, []byte(textResult.String()), 0644)
	if err != nil {
		fmt.Printf("Ошибка при сохранении файла transcription.txt: %v\n", err)
		return
	}
	fmt.Println("Общий текст сохранён в файл `transcription.txt`")
}
