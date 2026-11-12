package store

// The unqualified supertype below is the file's own nested class. A supertype
// written `extends java.io.OutputStream` is the JDK's, and must not be bound to
// it; the parser does not yet record a supertype's qualifier, so that line is
// not in this fixture (see internal/resolve/infer/supertype.go).
object Streams { // def sc.Streams
  class OutputStream { // def sc.Streams.OutputStream
  }

  class Logging extends OutputStream { // super OutputStream -> sc.Streams.OutputStream
  }
}
