package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Question struct {
	Type     string   `json:"type"`
	Question string   `json:"question"`
	Choices  []string `json:"choices,omitempty"`
	Answer   string   `json:"answer,omitempty"`
}

type CVItem struct {
	AudioPath string
	Sentence  string
	Level     string
}

var questions []Question
var cvItemsMap map[string][]CVItem
var currentCVItem *CVItem
var hintLevels map[string]int
var mistakeCounts map[string]int

// インドネシア語単語辞書（単語 -> [意味, 類義語]）
var wordDictionary = map[string][]string{
	"saya":                {"私", "aku, gue"},
	"kamu":                {"あなた", "anda, engkau"},
	"dia":                 {"彼/彼女", "ia"},
	"kita":                {"私たち", "kami"},
	"mereka":              {"彼ら", "orang-orang"},
	"apa":                 {"何", "apa itu"},
	"siapa":               {"誰", "orang siapa"},
	"dimana":              {"どこ", "di mana"},
	"kapan":               {"いつ", "bilamana"},
	"bagaimana":           {"どうやって", "cara apa"},
	"mengapa":             {"なぜ", "kenapa"},
	"ya":                  {"はい", "iya, betul"},
	"tidak":               {"いいえ", "enggak, tidak"},
	"makan":               {"食べる", "makan nasi"},
	"minum":               {"飲む", "minum air"},
	"pergi":               {"行く", "pergi ke"},
	"datang":              {"来る", "tiba"},
	"lihat":               {"見る", "melihat"},
	"dengar":              {"聞く", "mendengar"},
	"bicara":              {"話す", "berbicara"},
	"tulis":               {"書く", "menulis"},
	"baca":                {"読む", "membaca"},
	"besar":               {"大きい", "gede"},
	"kecil":               {"小さい", "kecilan"},
	"baik":                {"良い", "bagus"},
	"buruk":               {"悪い", "jelek"},
	"cepat":               {"速い", "laju"},
	"lambat":              {"遅い", "pelan"},
	"panas":               {"熱い", "hangat"},
	"dingin":              {"冷たい", "sejuk"},
	"rumah":               {"家", "kediaman"},
	"sekolah":             {"学校", "madrasah"},
	"kerja":               {"仕事", "pekerjaan"},
	"uang":                {"お金", "duit"},
	"air":                 {"水", "air minum"},
	"makanan":             {"食べ物", "hidangan"},
	"orang":               {"人", "manusia"},
	"anak":                {"子供", "anak kecil"},
	"ibu":                 {"母", "mama"},
	"ayah":                {"父", "papa"},
	"teman":               {"友達", "kawan"},
	"waktu":               {"時間", "masa"},
	"hari":                {"日", "tanggal"},
	"malam":               {"夜", "petang"},
	"pagi":                {"朝", "subuh"},
	"siang":               {"昼", "tengah hari"},
	"cinta":               {"愛", "kasih"},
	"suka":                {"好き", "senang"},
	"benci":               {"嫌い", "tidak suka"},
	"senang":              {"嬉しい", "gembira"},
	"sedih":               {"悲しい", "susah"},
	"marah":               {"怒る", "emosi"},
	"takut":               {"怖い", "gentar"},
	"bahagia":             {"幸せ", "riang"},
	"sakit":               {"病気", "penyakit"},
	"sehat":               {"健康", "fit"},
	"belajar":             {"勉強する", "mempelajari"},
	"mengajar":            {"教える", "mendidik"},
	"jalan":               {"歩く", "berjalan"},
	"lari":                {"走る", "berlari"},
	"duduk":               {"座る", "duduk"},
	"tidur":               {"寝る", "beristirahat"},
	"bangun":              {"起きる", "terbangun"},
	"mandi":               {"お風呂に入る", "membersihkan diri"},
	"pakai":               {"着る", "memakai"},
	"buka":                {"開ける", "membuka"},
	"tutup":               {"閉める", "menutup"},
	"masuk":               {"入る", "memasuki"},
	"keluar":              {"出る", "keluar"},
	"naik":                {"上がる", "menanjak"},
	"turun":               {"下がる", "menurun"},
	"kiri":                {"左", "sebelah kiri"},
	"kanan":               {"右", "sebelah kanan"},
	"atas":                {"上", "di atas"},
	"bawah":               {"下", "di bawah"},
	"depan":               {"前", "di depan"},
	"belakang":            {"後ろ", "di belakang"},
	"dalam":               {"中", "di dalam"},
	"luar":                {"外", "di luar"},
	"banyak":              {"多い", "banyak sekali"},
	"sedikit":             {"少ない", "sedikit sekali"},
	"semua":               {"全て", "seluruh"},
	"beberapa":            {"いくつか", "beberapa"},
	"pertama":             {"最初", "awal"},
	"terakhir":            {"最後", "akhir"},
	"baru":                {"新しい", "anyar"},
	"lama":                {"古い", "tua"},
	"hitam":               {"黒い", "gelap"},
	"putih":               {"白い", "pucat"},
	"merah":               {"赤い", "marun"},
	"biru":                {"青い", "nila"},
	"hijau":               {"緑", "daun"},
	"kuning":              {"黄色", "emas"},
	"cantik":              {"美しい", "indah"},
	"ganteng":             {"ハンサム", "tampan"},
	"jelek":               {"醜い", "buruk rupa"},
	"murah":               {"安い", "terjangkau"},
	"mahal":               {"高い", "berharga"},
	"jauh":                {"遠い", "distant"},
	"dekat":               {"近い", "terdekat"},
	"keras":               {"硬い", "tegar"},
	"lembut":              {"柔らかい", "halus"},
	"berat":               {"重い", "berbobot"},
	"ringan":              {"軽い", "enteng"},
	"panjang":             {"長い", "memanjang"},
	"pendek":              {"短い", "cepat"},
	"lebar":               {"広い", "lapang"},
	"sempit":              {"狭い", "sempit"},
	"tinggi":              {"高い", "menjulang"},
	"rendah":              {"低い", "menurun"},
	"kuat":                {"強い", "tangguh"},
	"lemah":               {"弱い", "tak berdaya"},
	"mudah":               {"簡単", "gampang"},
	"sulit":               {"難しい", "susah"},
	"benar":               {"正しい", "betul"},
	"salah":               {"間違っている", "keliru"},
	"bagus":               {"良い", "baik"},
	"indah":               {"美しい", "cantik"},
	"ramah":               {"親切", "sopan"},
	"jahat":               {"悪い", "nakal"},
	"pintar":              {"賢い", "cerdas"},
	"bodoh":               {"愚かな", "tolol"},
	"kaya":                {"富んでいる", "berduit"},
	"miskin":              {"貧しい", "fakir"},
	"bahasa":              {"言語", "lidah"},
	"kata":                {"言葉", "ucapan"},
	"kalimat":             {"文", "pernyataan"},
	"nama":                {"名前", "sebutan"},
	"alamat":              {"住所", "lokasi"},
	"nomor":               {"番号", "angka"},
	"telepon":             {"電話", "telp"},
	"surat":               {"手紙", "pesan"},
	"buku":                {"本", "kitab"},
	"kertas":              {"紙", "kertas tulis"},
	"pensil":              {"鉛筆", "potlot"},
	"pulpen":              {"ボールペン", "pena"},
	"meja":                {"机", "meja tulis"},
	"kursi":               {"椅子", "bangku"},
	"pintu":               {"ドア", "pintu masuk"},
	"jendela":             {"窓", "jendela kaca"},
	"lantai":              {"床", "dasar"},
	"dinding":             {"壁", "tembok"},
	"atap":                {"屋根", "genteng"},
	"kamar":               {"部屋", "ruangan"},
	"dapur":               {"キッチン", "kompor"},
	"kamar mandi":         {"お風呂場", "toilet"},
	"taman":               {"庭", "halaman"},
	"mobil":               {"車", "kendaraan"},
	"motor":               {"バイク", "sepeda motor"},
	"bis":                 {"バス", "bus"},
	"kereta":              {"電車", "train"},
	"pesawat":             {"飛行機", "airplane"},
	"kapal":               {"船", "perahu"},
	"sepeda":              {"自転車", "bike"},
	"makan siang":         {"昼食", "siang hari"},
	"makan malam":         {"夕食", "malam hari"},
	"sarapan":             {"朝食", "pagi hari"},
	"buah":                {"果物", "buah-buahan"},
	"sayur":               {"野菜", "sayuran"},
	"daging":              {"肉", "protein"},
	"ikan":                {"魚", "seafood"},
	"ayam":                {"鶏肉", "unggas"},
	"nasi":                {"ご飯", "beras"},
	"roti":                {"パン", "bread"},
	"susu":                {"牛乳", "milk"},
	"kopi":                {"コーヒー", "coffee"},
	"teh":                 {"お茶", "tea"},
	"jus":                 {"ジュース", "juice"},
	"air mineral":         {"ミネラルウォーター", "mineral water"},
	"mie":                 {"麺", "noodle"},
	"soto":                {"ソト（スープ）", "sup ayam"},
	"nasi goreng":         {"チャーハン", "fried rice"},
	"rendang":             {"レンダン（肉料理）", "daging masak"},
	"gado-gado":           {"ガドガド（野菜サラダ）", "salad sayur"},
	"bakso":               {"肉団子", "bola daging"},
	"martabak":            {"マルタバク（お菓子）", "kue manis"},
	"pisang":              {"バナナ", "banana"},
	"apel":                {"リンゴ", "apple"},
	"jeruk":               {"オレンジ", "orange"},
	"mangga":              {"マンゴー", "mango"},
	"semangka":            {"スイカ", "watermelon"},
	"anggur":              {"ブドウ", "grape"},
	"stroberi":            {"イチゴ", "strawberry"},
	"durian":              {"ドリアン", "buah durian"},
	"salak":               {"サラッ（果物）", "buah salak"},
	"rambutan":            {"ランブータン", "buah rambutan"},
	"kelapa":              {"ココナッツ", "coconut"},
	"wortel":              {"ニンジン", "carrot"},
	"kentang":             {"ジャガイモ", "potato"},
	"tomat":               {"トマト", "tomato"},
	"bawang":              {"玉ねぎ", "onion"},
	"cabe":                {"唐辛子", "chili"},
	"kol":                 {"キャベツ", "cabbage"},
	"sawah":               {"田んぼ", "ladang"},
	"gunung":              {"山", "pegunungan"},
	"sungai":              {"川", "aliran air"},
	"laut":                {"海", "samudera"},
	"pulau":               {"島", "nusa"},
	"hutan":               {"森", "rimba"},
	"desa":                {"村", "kampung"},
	"kota":                {"都市", "metropolitan"},
	"negara":              {"国", "bangsa"},
	"dunia":               {"世界", "bumi"},
	"bumi":                {"地球", "dunia"},
	"langit":              {"空", "udara"},
	"bulan":               {"月", "satellite"},
	"bintang":             {"星", "asteroid"},
	"matahari":            {"太陽", "sun"},
	"hujan":               {"雨", "curah hujan"},
	"salju":               {"雪", "snow"},
	"angin":               {"風", "hembusan"},
	"awan":                {"雲", "mendung"},
	"petir":               {"雷", "halilintar"},
	"pelangi":             {"虹", "rainbow"},
	"musim":               {"季節", "season"},
	"kering":              {"乾季", "musim kemarau"},
	"musim semi":          {"春", "spring"},
	"musim panas":         {"夏", "summer"},
	"musim gugur":         {"秋", "autumn"},
	"musim dingin":        {"冬", "winter"},
	"tahun":               {"年", "periode"},
	"minggu":              {"週", "pekan"},
	"jam":                 {"時間", "waktu"},
	"menit":               {"分", "minute"},
	"detik":               {"秒", "second"},
	"sekarang":            {"今", "kini"},
	"kemarin":             {"昨日", "hari lalu"},
	"besok":               {"明日", "hari depan"},
	"lusa":                {"明後日", "dua hari lagi"},
	"minggu lalu":         {"先週", "pekan lalu"},
	"minggu depan":        {"来週", "pekan depan"},
	"bulan lalu":          {"先月", "bulan kemarin"},
	"bulan depan":         {"来月", "bulan mendatang"},
	"tahun lalu":          {"去年", "tahun kemarin"},
	"tahun depan":         {"来年", "tahun mendatang"},
	"ulang tahun":         {"誕生日", "hari jadi"},
	"libur":               {"休日", "hari libur"},
	"kuliah":              {"大学", "perguruan tinggi"},
	"ujian":               {"試験", "tes"},
	"nilai":               {"成績", "skor"},
	"guru":                {"先生", "pengajar"},
	"murid":               {"生徒", "siswa"},
	"dosen":               {"講師", "pengajar tinggi"},
	"mahasiswa":           {"大学生", "pelajar tinggi"},
	"kantor":              {"オフィス", "tempat kerja"},
	"pabrik":              {"工場", "industri"},
	"toko":                {"店", "warung"},
	"pasar":               {"市場", "tempat jual beli"},
	"bank":                {"銀行", "lembaga keuangan"},
	"rumah sakit":         {"病院", "klinik"},
	"apotek":              {"薬局", "farmasi"},
	"polisi":              {"警察", "kepolisian"},
	"pemadam kebakaran":   {"消防", "fire brigade"},
	"pos":                 {"郵便局", "kantor pos"},
	"bioskop":             {"映画館", "cinema"},
	"restoran":            {"レストラン", "rumah makan"},
	"hotel":               {"ホテル", "penginapan"},
	"stadion":             {"スタジアム", "arena olahraga"},
	"lapangan":            {"グラウンド", "field"},
	"kolam renang":        {"プール", "swimming pool"},
	"gym":                 {"ジム", "pusat kebugaran"},
	"museum":              {"博物館", "museum"},
	"perpustakaan":        {"図書館", "library"},
	"gereja":              {"教会", "tempat ibadah"},
	"masjid":              {"モスク", "tempat sholat"},
	"pura":                {"ヒンドゥー寺院", "tempat sembahyang"},
	"vihara":              {"仏教寺院", "tempat meditasi"},
	"keluarga":            {"家族", "household"},
	"saudara":             {"兄弟姉妹", "siblings"},
	"kakak":               {"兄姉", "abang/kakak"},
	"adik":                {"弟妹", "adek"},
	"paman":               {"叔父", "om"},
	"bibi":                {"叔母", "tante"},
	"kakek":               {"祖父", "nenek"},
	"nenek":               {"祖母", "kakek"},
	"cucu":                {"孫", "grandchild"},
	"suami":               {"夫", "istri"},
	"istri":               {"妻", "suami"},
	"pacar":               {"恋人", "kekasih"},
	"tunangan":            {"婚約者", "calon suami/istri"},
	"janda":               {"未亡人", "duda"},
	"duda":                {"未亡人", "janda"},
	"anak yatim":          {"孤児", "anak tanpa orang tua"},
	"anak angkat":         {"養子", "adopted child"},
	"adik angkat":         {"義理の兄弟", "saudara angkat"},
	"teman baik":          {"親友", "best friend"},
	"kenalan":             {"知り合い", "acquaintance"},
	"tetangga":            {"隣人", "neighbor"},
	"rekan kerja":         {"同僚", "colleague"},
	"atasan":              {"上司", "bos"},
	"bawahan":             {"部下", "karyawan"},
	"pelanggan":           {"顧客", "customer"},
	"penjual":             {"売り手", "seller"},
	"pembeli":             {"買い手", "buyer"},
	"supir":               {"運転手", "driver"},
	"masinis":             {"機関士", "engineer"},
	"pilot":               {"パイロット", "pilot"},
	"pramugari":           {"キャビンアテンダント", "flight attendant"},
	"dokter":              {"医者", "physician"},
	"perawat":             {"看護師", "nurse"},
	"apoteker":            {"薬剤師", "pharmacist"},
	"pengacara":           {"弁護士", "lawyer"},
	"hakim":               {"裁判官", "judge"},
	"tentara":             {"兵士", "soldier"},
	"nelayan":             {"漁師", "fisherman"},
	"petani":              {"農民", "farmer"},
	"buruh":               {"労働者", "worker"},
	"pegawai":             {"公務員", "civil servant"},
	"guru besar":          {"教授", "professor"},
	"peneliti":            {"研究者", "researcher"},
	"penulis":             {"作家", "author"},
	"wartawan":            {"記者", "journalist"},
	"fotografer":          {"写真家", "photographer"},
	"aktor":               {"俳優", "actress"},
	"penyanyi":            {"歌手", "singer"},
	"musisi":              {"ミュージシャン", "musician"},
	"pelukis":             {"画家", "painter"},
	"pematung":            {"彫刻家", "sculptor"},
	"penari":              {"ダンサー", "dancer"},
	"atlet":               {"アスリート", "athlete"},
	"olahraga":            {"スポーツ", "sport"},
	"sepak bola":          {"サッカー", "football"},
	"basket":              {"バスケットボール", "basketball"},
	"voli":                {"バレーボール", "volleyball"},
	"tenis":               {"テニス", "tennis"},
	"badminton":           {"バドミントン", "badminton"},
	"golf":                {"ゴルフ", "golf"},
	"renang":              {"水泳", "swimming"},
	"angkat besi":         {"重量挙げ", "weightlifting"},
	"tinju":               {"ボクシング", "boxing"},
	"silat":               {"シラット（インドネシア武術）", "pencak silat"},
	"bulu tangkis":        {"バドミントン", "badminton"},
	"sepak takraw":        {"セパックタクロー", "kick volleyball"},
	"panahan":             {"弓道", "archery"},
	"balap":               {"レース", "racing"},
	"balap motor":         {"バイクレース", "motorbike racing"},
	"balap mobil":         {"カーレース", "car racing"},
	"f1":                  {"F1", "formula 1"},
	"motogp":              {"MotoGP", "motor grand prix"},
	"tour de france":      {"ツール・ド・フランス", "tour de france"},
	"olimpiade":           {"オリンピック", "olympics"},
	"piala dunia":         {"ワールドカップ", "world cup"},
	"liga":                {"リーグ", "league"},
	"tim":                 {"チーム", "team"},
	"pemain":              {"選手", "player"},
	"pelatih":             {"コーチ", "coach"},
	"wasit":               {"審判", "referee"},
	"gol":                 {"ゴール", "goal"},
	"poin":                {"ポイント", "point"},
	"menang":              {"勝つ", "win"},
	"kalah":               {"負ける", "lose"},
	"seri":                {"引き分け", "draw"},
	"final":               {"決勝", "final"},
	"semifinal":           {"準決勝", "semifinal"},
	"perempat final":      {"準々決勝", "quarterfinal"},
	"grup":                {"グループ", "group"},
	"babak":               {"ラウンド", "round"},
	"turnamen":            {"トーナメント", "tournament"},
	"kompetisi":           {"競技", "competition"},
	"pertandingan":        {"試合", "match"},
	"kejurnas":            {"全国選手権", "national championship"},
	"liga champions":      {"チャンピオンズリーグ", "champions league"},
	"premier league":      {"プレミアリーグ", "premier league"},
	"serie a":             {"セリエA", "serie a"},
	"la liga":             {"ラ・リーガ", "la liga"},
	"bundesliga":          {"ブンデスリーガ", "bundesliga"},
	"liga 1":              {"リーグ1", "league 1"},
	"eredivisie":          {"エールディヴィジ", "eredivisie"},
	"primeira liga":       {"プリメイラ・リーガ", "primeira liga"},
	"ligue 1":             {"リーグ・アン", "ligue 1"},
	"mls":                 {"MLS", "major league soccer"},
	"j league":            {"Jリーグ", "j league"},
	"liga indonesia":      {"インドネシアリーグ", "indonesian league"},
	"persib":              {"ペルシプ", "persib bandung"},
	"persija":             {"ペルシジャ", "persija jakarta"},
	"arema":               {"アレマ", "arema malang"},
	"persebaya":           {"ペルセバヤ", "persebaya surabaya"},
	"mitra kukar":         {"ミトラ・クカル", "mitra kukar"},
	"borneo fc":           {"ボルネオFC", "borneo fc"},
	"bali united":         {"バリ・ユナイテッド", "bali united"},
	"psm makassar":        {"PSMマカッサル", "psm makassar"},
	"persipura":           {"ペルシパラ", "persipura jayapura"},
	"ps tni":              {"PS TNI", "ps tni"},
	"psms medan":          {"PSMSメダン", "psms medan"},
	"semen padang":        {"セメン・パダン", "semen padang"},
	"barito putera":       {"バリト・プテラ", "barito putera"},
	"bhayangkara":         {"バヤンカラ", "bhayangkara fc"},
	"kalteng putra":       {"カルテン・プトラ", "kalteng putra"},
	"persela lamongan":    {"ペルセラ・ラモングン", "persela lamongan"},
	"perseru serui":       {"ペルセル・セルイ", "perseru serui"},
	"persiba balikpapan":  {"ペルシバ・バリクパパン", "persiba balikpapan"},
	"persiwa wamena":      {"ペルシワ・ワメナ", "persiwa wamena"},
	"persikabo 1973":      {"ペルシカボ1973", "persikabo 1973"},
	"persis solo":         {"ペルシス・ソロ", "persis solo"},
	"persita tanggerang":  {"ペルシタ・タンゲラン", "persita tanggerang"},
	"persik kediri":       {"ペルシク・ケディリ", "persik kediri"},
	"persija jakarta":     {"ペルシジャ・ジャカルタ", "persija jakarta"},
	"persib bandung":      {"ペルシプ・バンドン", "persib bandung"},
	"arema fc":            {"アレマFC", "arema fc"},
	"persebaya surabaya":  {"ペルセバヤ・スラバヤ", "persebaya surabaya"},
	"mitra kukar fc":      {"ミトラ・クカルFC", "mitra kukar fc"},
	"borneo fc samarinda": {"ボルネオFCサマリンダ", "borneo fc samarinda"},
	"bali united fc":      {"バリ・ユナイテッドFC", "bali united fc"},
	"psm fc":              {"PSM FC", "psm fc"},
	"persipura jayapura":  {"ペルシパラ・ジャヤプラ", "persipura jayapura"},
}

func getWordInfo(word string) (string, string) {
	if info, exists := wordDictionary[strings.ToLower(word)]; exists {
		return info[0], info[1] // 意味, 類義語
	}
	return "不明", "該当なし"
}

func formatWordInfo(sentence string) string {
	words := strings.Fields(sentence)
	var result []string

	for _, word := range words {
		// 句読点を取り除く
		cleanWord := strings.TrimRight(word, ".,!?")
		meaning, synonyms := getWordInfo(cleanWord)
		result = append(result, fmt.Sprintf("%s: %s (%s)", cleanWord, meaning, synonyms))
	}

	return strings.Join(result, "\n")
}

func loadQuestions() {
	data, err := os.ReadFile("questions.json")
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(data, &questions)
}

func LoadCommonVoice(tsvPath string) ([]CVItem, error) {
	file, err := os.Open(tsvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	log.Printf("Read %d records from %s", len(records), tsvPath)

	var items []CVItem
	for i, r := range records {
		if i == 0 {
			continue // header
		}
		if len(r) < 4 {
			continue
		}
		sentence := r[3]
		words := strings.Fields(sentence)

		if len(words) < 3 {
			continue
		}

		level := "normal"
		switch {
		case len(words) <= 5:
			level = "easy"
		case len(words) >= 10:
			level = "hard"
		}

		items = append(items, CVItem{
			AudioPath: "mcv-scripted-id-v24.0/cv-corpus-24.0-2025-12-05/id/clips/" + r[1],
			Sentence:  sentence,
			Level:     level,
		})
	}

	return items, nil
}

func loadCVItems() {
	items, err := LoadCommonVoice("mcv-scripted-id-v24.0/cv-corpus-24.0-2025-12-05/id/validated.tsv")
	if err != nil {
		log.Printf("Failed to load CV items: %v", err)
		return
	}
	cvItemsMap = make(map[string][]CVItem)
	for _, item := range items {
		cvItemsMap[item.Level] = append(cvItemsMap[item.Level], item)
	}
	log.Printf("Loaded CV items: easy=%d, normal=%d, hard=%d", len(cvItemsMap["easy"]), len(cvItemsMap["normal"]), len(cvItemsMap["hard"]))
}

func Normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, ",", "")
	return strings.TrimSpace(s)
}

func Check(user, answer string) bool {
	return Normalize(user) == Normalize(answer)
}

func getMatchedWords(user, answer string) []string {
	u := Normalize(user)
	a := Normalize(answer)
	uwords := strings.Fields(u)
	awords := strings.Fields(a)
	seen := make(map[string]bool)
	set := make(map[string]bool)
	for _, w := range awords {
		seen[w] = true
	}
	var matched []string
	for _, w := range uwords {
		if seen[w] && !set[w] {
			matched = append(matched, w)
			set[w] = true
		}
	}
	return matched
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN が設定されていません")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	rand.Seed(time.Now().UnixNano())
	loadQuestions()
	loadCVItems()
	hintLevels = make(map[string]int)
	mistakeCounts = make(map[string]int)

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.Bot {
			return
		}

		if m.Content == "!ping" {
			s.ChannelMessageSend(m.ChannelID, "pong")
		}

		if strings.HasPrefix(m.Content, "!cv") {
			// 前の問題が未解決なら答えを表示
			if currentCVItem != nil {
				s.ChannelMessageSend(m.ChannelID, "前の問題が未解決でした。正解は: "+currentCVItem.Sentence)
			}
			hintLevels[m.Author.ID] = 0
			parts := strings.Fields(m.Content)
			level := "all"
			if len(parts) > 1 {
				level = parts[1]
			}
			var selectedItems []CVItem
			if level == "all" {
				for _, items := range cvItemsMap {
					selectedItems = append(selectedItems, items...)
				}
			} else {
				selectedItems = cvItemsMap[level]
			}
			if len(selectedItems) == 0 {
				s.ChannelMessageSend(m.ChannelID, "No CV items loaded for level: "+level)
				return
			}
			item := selectedItems[rand.Intn(len(selectedItems))]
			currentCVItem = &item
			file, err := os.Open(item.AudioPath)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Error opening audio file")
				return
			}
			defer file.Close()
			s.ChannelFileSend(m.ChannelID, "listening.mp3", file)
			s.ChannelMessageSend(m.ChannelID, "Listen to the audio and type the sentence!")
		}

		if currentCVItem != nil && !strings.HasPrefix(m.Content, "!") {
			userInput := m.Content
			userID := m.Author.ID
			if Check(userInput, currentCVItem.Sentence) {
				wordInfo := formatWordInfo(currentCVItem.Sentence)
				response := "Correct! 🎉\n\n単語情報:\n" + wordInfo
				s.ChannelMessageSend(m.ChannelID, response)
				mistakeCounts[userID] = 0
				currentCVItem = nil
				return
			}
			// 部分一致の単語を抽出
			matched := getMatchedWords(userInput, currentCVItem.Sentence)
			mistakeCounts[userID]++
			if len(matched) > 0 {
				msg := "部分一致した単語: " + strings.Join(matched, ", ") + "\n"
				if mistakeCounts[userID] >= 3 {
					wordInfo := formatWordInfo(currentCVItem.Sentence)
					msg += "不正解。正解は: " + currentCVItem.Sentence + "\n\n単語情報:\n" + wordInfo
					s.ChannelMessageSend(m.ChannelID, msg)
					mistakeCounts[userID] = 0
					currentCVItem = nil
					return
				}
				remain := 3 - mistakeCounts[userID]
				msg += fmt.Sprintf("まだ不正解です。残り試行回数: %d", remain)
				s.ChannelMessageSend(m.ChannelID, msg)
			} else {
				if mistakeCounts[userID] >= 3 {
					wordInfo := formatWordInfo(currentCVItem.Sentence)
					s.ChannelMessageSend(m.ChannelID, "不正解。正解は: "+currentCVItem.Sentence+"\n\n単語情報:\n"+wordInfo)
					mistakeCounts[userID] = 0
					currentCVItem = nil
					return
				}
				remain := 3 - mistakeCounts[userID]
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("不正解です。残り試行回数: %d", remain))
			}
		}

		if m.Content == "!answer" {
			if currentCVItem == nil {
				s.ChannelMessageSend(m.ChannelID, "No current CV item. Use !cv first.")
				return
			}
			userID := m.Author.ID
			wordInfo := formatWordInfo(currentCVItem.Sentence)
			response := "回答: " + currentCVItem.Sentence + "\n\n単語情報:\n" + wordInfo
			s.ChannelMessageSend(m.ChannelID, response)
			mistakeCounts[userID] = 0
			hintLevels[userID] = 0
			currentCVItem = nil
		}

		if m.Content == "!hint" {
			if currentCVItem == nil {
				s.ChannelMessageSend(m.ChannelID, "No current CV item. Use !cv first.")
				return
			}
			userID := m.Author.ID
			level := hintLevels[userID]
			words := strings.Fields(currentCVItem.Sentence)
			var hint string
			switch level {
			case 0:
				hint = fmt.Sprintf("単語数: %d", len(words))
			case 1:
				charCounts := make([]string, len(words))
				charHints := make([]string, len(words))
				for i, w := range words {
					charCounts[i] = strconv.Itoa(len(w))
					charHints[i] = strings.Repeat("\\_", len(w))
				}
				hint = "単語の文字数: " + strings.Join(charCounts, ", ") + " " + strings.Join(charHints, " ")
			case 2:
				// 仮定の品詞: 全て名詞として
				pos := make([]string, len(words))
				for i := range pos {
					pos[i] = "名詞"
				}
				hint = "品詞: " + strings.Join(pos, ", ")
			case 3:
				initialHints := make([]string, len(words))
				for i, w := range words {
					if len(w) > 0 {
						initialHints[i] = string(w[0]) + strings.Repeat("\\_", len(w)-1)
					}
				}
				hint = "単語の冒頭: " + strings.Join(initialHints, " ")
			default:
				revealLevel := level - 3
				initialHints := make([]string, len(words))
				for i, w := range words {
					if len(w) > 0 {
						initialHints[i] = string(w[0]) + strings.Repeat("\\_", len(w)-1)
					}
				}
				if revealLevel < len(words) {
					hint = "最初の " + strconv.Itoa(revealLevel) + " 単語: " + strings.Join(words[:revealLevel], " ") + " " + strings.Join(initialHints[revealLevel:], " ")
				} else {
					hint = "全ての文が出ました 答え: " + currentCVItem.Sentence
					currentCVItem = nil
				}
			}
			s.ChannelMessageSend(m.ChannelID, hint)
			hintLevels[userID]++
		}

		if m.Content == "!today" {
			q := questions[rand.Intn(len(questions))]

			msg := "📘 今日の一問\n" + q.Question

			if q.Type == "vocab" && len(q.Choices) > 0 {
				for i, c := range q.Choices {
					msg += "\n" + string('A'+i) + ". " + c
				}
			}

			s.ChannelMessageSend(m.ChannelID, msg)
		}

	})

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot is running")

	// 終了待ち
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	dg.Close()
}
