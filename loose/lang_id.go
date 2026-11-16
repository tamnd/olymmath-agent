package loose

func init() {
	// Indonesian, for the Indonesian olympiad sets. Its common words are not common words of the other languages that Lang tells apart.
	languages["indonesian"], languages["id"] = "id", "id"
	stopWords["id"] = []string{"yang", "dan", "adalah", "dari", "untuk", "dengan", "jika", "maka", "pada", "di", "ini", "itu", "bilangan", "tentukan", "semua", "banyaknya", "nilai", "sehingga", "merupakan", "berapa"}
}
