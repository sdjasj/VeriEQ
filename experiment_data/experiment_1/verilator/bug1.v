`timescale 1ns/1ps
module a();
reg signed [2:0] in0;

initial begin
  in0 = 7; // signed 3'b111
  #1;
  $display("in0: %0d", in0);
  case (in0)
    // compare signed 3'b111 and unsigned 4'b0111
    // signed 3'b111 ----> unsigned 4'b0111
    // so 4'b0111 == 4'b0111, match
    4'b111: begin
      $display("1");
    end
    default: begin
      $display("2");
    end
  endcase

end

endmodule