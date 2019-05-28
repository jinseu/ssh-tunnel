func getPublicSuffix(domain string) {
	tld, _ := publicsuffix.EffectiveTLDPlusOne(domain)
	fmt.Printf("EffectiveTLDPlusOne: %s\n", tld)
	suffix, _ := publicsuffix.PublicSuffix(domain)
	fmt.Printf("PublicSuffix: %s\n", suffix)
}

func reload(configFile string) {
	file, err := NewConfigFile(os.ExpandEnv(configFile))
	if err != nil {
		L.Fatal(err)
	}
	res, err := http.Get(fmt.Sprintf("http://%s/reload", file.LocalNormalServer))
	if err != nil {
		L.Fatal(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		L.Fatal(err)
	}
	fmt.Printf("%s\n", body)
}

func main(){
	publicSuffix := flag.String("suffix", "", "print pulbic suffix for the given domain")
	reload       := flag.Bool("reload", false, "send signal to reload config file")


	if *FSuffix != "" {
		getPublicSuffix(*publicSuffix)
	} else if *FReload {
		reload(*configFile)
	} else {
		serve(*configFile)
	}
}