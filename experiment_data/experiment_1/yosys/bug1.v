module a(input b, c, output reg d);
  reg [31:0] e;

  always @(*) begin
    if (b >> e) begin
        d = e;
    end

    if (c - c) begin
      e = 0;
    end

    e = !e;
  end
endmodule