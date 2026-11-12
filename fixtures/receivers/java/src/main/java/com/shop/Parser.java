package com.shop;

import com.shop.io.JsonReader;
import java.math.BigDecimal;
import java.util.List;

public class Parser extends Base {
  private JsonReader reader(String json) { // def Parser.reader
    return new JsonReader();
  }

  public int intValue() { // def Parser.intValue
    return asBigDecimal().intValue(); // call intValue -> none
  }

  private BigDecimal asBigDecimal() { // def Parser.asBigDecimal
    return BigDecimal.ONE;
  }

  public int parse(String json, List<String> strings) {
    JsonReader reader = reader(json); // call reader -> Parser.reader high
    strings.add(json); // call add -> external; nocall strings
    return reader.peek(); // call peek -> JsonReader.peek high; nocall reader
  }

  @Override
  public String describe() { // def Parser.describe
    return super.describe(); // call describe -> Base.describe high
  }

  @Override
  protected Object clone() throws CloneNotSupportedException { // def Parser.clone
    return super.clone(); // call clone -> none
  }
}
