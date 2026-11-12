# `include Mod` in a class body is Module#include. rack bound it 18 times to the
# one `include` the repository declares.
class Recursive # def rb.Recursive
  def include(app) # def rb.Recursive.include
    app
  end
end

class Mixed # def rb.Mixed
  include Enumerable # call include -> none
  extend Comparable # call extend -> none
end
