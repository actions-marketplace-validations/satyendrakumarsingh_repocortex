package infer

// # Bare names a language's own library provides
//
// corenames.go lists the MEMBER names a language's core types declare, so that
// `x.toString()` is not bound to the one `toString` a repository declares. This
// is the same argument for a call written with no receiver: `require(x)` in
// Kotlin, `fgets(buf, n, f)` in C, `include Mod` in a Ruby class body and
// `format!(...)` in Rust name something the language or its standard library
// provides, and a repository that happens to declare ONE function of that name
// somewhere says nothing about which a call reaches. The other definitions are
// in a library that was never indexed.
//
// The principle is the one the member lists state: the language's standard
// library and its dominant test framework, nothing wider and not tuned to any
// project. What a name here costs is the correct binding of a repository
// function that shares it, called from a file that neither defines it nor
// imports it. A definition in the caller's own file, or in the type it is
// written in, or one the file imports by name, is evidence about THIS call, and
// is never refused on account of a list. Only the rules that pick a target by
// its name alone are (see bareByName and bareBeyondType).

// coreFunctions maps a language to the bare names its library provides.
var coreFunctions = buildCoreFunctions()

// coreSyntax maps a language to names that are invoked with a bare call and
// can never be a repository's function: Rust's standard macros. The parser
// records `format!(x)` as a call to `format` with no receiver, and a macro
// namespace is separate from the function one, so a function that shares the
// name is not what the invocation reaches even in the same file. Measured on
// fd, where `format!` was bound at `high` to a test function named `format`.
//
// Kept to names whose use as a function is the exception. `write`, `print`,
// `matches`, `file`, `line` and the like are macros too, and are in
// coreFunctions instead: refused to a name-only rule, allowed to a definition
// the file itself holds.
var coreSyntax = map[string]map[string]struct{}{
	"rust": words(`
		println eprintln eprint format format_args vec panic assert assert_eq assert_ne
		debug_assert debug_assert_eq debug_assert_ne todo unimplemented unreachable
		writeln dbg`),
}

func buildCoreFunctions() map[string]map[string]struct{} {
	out := make(map[string]map[string]struct{}, 8)

	// Rust: the macros that are also plausible function names.
	out["rust"] = words(`
		write print matches file line column env option_env concat stringify include
		include_str include_bytes cfg module_path compile_error thread_local`)

	// Kotlin: kotlin.* and kotlin.collections.*, which every file imports
	// implicitly, and the constructors of the JVM types a Kotlin file names
	// without an import. Measured on moshi, where `require(...)` and `Pair(...)`
	// were bound to a private member of an unrelated class and to a Java test
	// class.
	out["kotlin"] = words(`
		require check error assert requireNotNull checkNotNull TODO
		listOf mutableListOf arrayListOf setOf mutableSetOf hashSetOf linkedSetOf
		mapOf mutableMapOf hashMapOf linkedMapOf sortedMapOf emptyList emptySet emptyMap
		emptySequence sequenceOf arrayOf arrayOfNulls intArrayOf longArrayOf byteArrayOf
		charArrayOf booleanArrayOf doubleArrayOf floatArrayOf shortArrayOf
		lazy lazyOf println print readLine readln repeat run with apply let also
		buildString buildList buildMap buildSet sequence generateSequence synchronized
		runCatching maxOf minOf Pair Triple Regex Result
		String StringBuilder Array List Map Set Any Unit Exception RuntimeException
		IllegalArgumentException IllegalStateException UnsupportedOperationException
		NoSuchElementException IndexOutOfBoundsException AssertionError Error Throwable
		ArrayList HashMap HashSet LinkedHashMap LinkedHashSet`)
	out["scala"] = words(`
		require assert assume println print Some None Left Right Option Seq List Map Set
		Vector Array String StringBuilder Try Success Failure Future Promise Tuple2
		Tuple3 Ordering`)

	// Ruby: Kernel and Module, called with no receiver in a class body or a
	// method. Measured on rack, where `include BadRequest` was bound 18 times to
	// the one `include` the repository declares (Rack::Recursive#include).
	out["ruby"] = words(`
		include extend prepend require require_relative load attr_reader attr_writer
		attr_accessor private public protected module_function alias_method
		raise puts print p pp format sprintf printf loop lambda proc catch throw
		block_given? binding freeze at_exit gets sleep rand srand exit abort warn fail
		open system exec fork trap Integer Float String Array Hash Rational Complex`)

	// C and C++: the C library, which a repository may wrap or interpose (jq's
	// fault-injection `fgets`) and a test calls through to.
	libc := words(`
		fgets fputs fopen fclose fread fwrite fprintf printf sprintf snprintf vsnprintf
		sscanf scanf fscanf puts putchar getc getchar fgetc fputc fflush fseek ftell
		rewind feof ferror perror setvbuf malloc calloc realloc free memcpy memmove
		memset memcmp memchr strlen strcmp strncmp strcpy strncpy strcat strncat strchr
		strrchr strstr strtok strdup strerror strtol strtoul strtod strtoll atoi atol
		atof abs labs qsort bsearch exit abort atexit getenv setenv unsetenv putenv
		system rand srand time clock difftime mktime gmtime localtime strftime sleep
		usleep nanosleep open close read write lseek stat fstat lstat isatty dup dup2
		pipe fork wait waitpid kill signal raise getpid unlink rename mkdir rmdir
		chdir getcwd opendir readdir closedir mmap munmap select poll socket bind
		listen accept connect send recv sin cos tan sqrt pow floor ceil fabs fmod log
		exp isalpha isdigit isspace isupper islower toupper tolower assert`)
	out["c"] = libc
	out["cpp"] = libc

	// ECMAScript: the platform's own constructors and namespaces, which a bare
	// `new URL(...)` or `Response(...)` names whether or not the repository
	// declares a class of the same name elsewhere.
	script := make(map[string]struct{}, len(scriptBuiltins))
	for _, b := range scriptBuiltins {
		script[b] = struct{}{}
	}
	for _, lang := range []string{"javascript", "typescript", "tsx", "vue", "svelte", "astro"} {
		out[lang] = script
	}
	return out
}

// isCoreFunction reports whether the language's own library provides a
// callable of this bare name.
func (s LanguageSpec) isCoreFunction(name string) bool {
	_, ok := coreFunctions[s.Name][name]
	return ok
}

// isCoreSyntax reports whether a bare call of this name is the language's own
// syntax, so that nothing the repository defines is what it reaches.
func (s LanguageSpec) isCoreSyntax(name string) bool {
	_, ok := coreSyntax[s.Name][name]
	return ok
}
