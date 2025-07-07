module test;

reg signed in4;
initial begin
  in4 = 1;

  #1;

  $display("in4: %0d", in4);

  case (0 ? 1'h0 : in4)
    5'b0101: begin
        $display("1");
    end

    8'b000001: begin
      $display("2");
    end

    default: begin
      $display("3");
    end
  endcase
end

endmodule